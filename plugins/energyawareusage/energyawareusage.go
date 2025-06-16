package energyawareusage

import (
    "context"
    "fmt"
    "math"
	"strconv"

    v1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/runtime"
    framework "k8s.io/kubernetes/pkg/scheduler/framework"
    fwk "k8s.io/kube-scheduler/framework"
)

const Name = "EnergyAwareUsage"

type EnergyAwareUsage struct {
	handle framework.Handle
}

var _ framework.ScorePlugin = &EnergyAwareUsage{}

func New(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
    fmt.Println("⚠️ EnergyAwareUsage NEW() CALLED!")
    return &EnergyAwareUsage{}, nil
}

func (v *EnergyAwareUsage) Name() string {
    return Name
}

func getPodResourceRequest(pod *v1.Pod) *framework.Resource {
    result := &framework.Resource{}
    for _, c := range pod.Spec.Containers {
        result.MilliCPU += c.Resources.Requests.Cpu().MilliValue()
        result.Memory += c.Resources.Requests.Memory().Value()
    }
    for _, c := range pod.Spec.InitContainers {
        cpu := c.Resources.Requests.Cpu().MilliValue()
        mem := c.Resources.Requests.Memory().Value()
        if cpu > result.MilliCPU {
            result.MilliCPU = cpu
        }
        if mem > result.Memory {
            result.Memory = mem
        }
    }
    return result
}

// Στο Plugin αυτό υπολογίζουμε την ενέργεια που καταναλώθηκε και πόσο sustainable είναι ο κόμβος βάση δεικτών για προβλεπόμενη ενεργειακή κατανάλωση.
func (v *EnergyAwareUsage) Score(
    ctx context.Context,
    state fwk.CycleState,
    pod *v1.Pod,
    nodeInfo *framework.NodeInfo,
) (int64, *framework.Status) {
	allocatableMilliCPU := float64(nodeInfo.Allocatable.MilliCPU)

	requestedMilliCPU := float64(nodeInfo.Requested.MilliCPU)

	podReq := getPodResourceRequest(pod)
	podReqMilliCPU := float64(podReq.MilliCPU)

	usedMilliCPU := requestedMilliCPU + podReqMilliCPU
	cpuUtil := 0.0
	if allocatableMilliCPU > 0 {
		cpuUtil = usedMilliCPU / allocatableMilliCPU
	}

	// Προεπιλεγμένες τιμές
	greenFactor := 1.0
	carbonPenalty := 1.0
	carbonIntensity := 1.0

	// Διαβάζουμε τα annotations που προσθέσαμε στους κόμβους
	if gfStr, ok := nodeInfo.Node().Annotations["greenFactor"]; ok {
		if val, err := strconv.ParseFloat(gfStr, 64); err == nil {
			greenFactor = val
		}
	}
	if cpStr, ok := nodeInfo.Node().Annotations["carbonPenalty"]; ok {
		if val, err := strconv.ParseFloat(cpStr, 64); err == nil {
			carbonPenalty = val
		}
	}

	// Διαβάζουμε τα annotations που προσθέσαμε στα Pods
	if ciStr, ok := pod.Annotations["carbonIntensity"]; ok {
		if val, err := strconv.ParseFloat(ciStr, 64); err == nil {
			carbonIntensity = val
		}
	}

	// Υπολογίζουμε η τελική αξιολόγηση βιωσιμότητας
	rawScore := (cpuUtil * greenFactor) - (carbonPenalty * carbonIntensity)

	// Κανονικοποιούμε το rawScore στο 1-100
	// Θεωρούμε πως οι πιθανές τιμές είναι μεταξύ -2 και 2 βάσει υπολογισμών στα μηχανήματα.
	minRaw := -2.0
	maxRaw := 2.0
	clamped := math.Max(minRaw, math.Min(maxRaw, rawScore))
	normalized := (clamped - minRaw) / (maxRaw - minRaw)
	finalScore := int64(1 + normalized*99)

	fmt.Printf("EnergyAwareUsage - Scoring Node %s: cpuUtil=%.2f, greenFactor=%.2f, carbonPenalty=%.2f, carbonIntensity=%.2f → Score = %d\n",
		nodeInfo.GetName(), cpuUtil, greenFactor, carbonPenalty, carbonIntensity, finalScore)

	return finalScore, framework.NewStatus(framework.Success)
}

func (v *EnergyAwareUsage) ScoreExtensions() framework.ScoreExtensions {
    return nil
}
