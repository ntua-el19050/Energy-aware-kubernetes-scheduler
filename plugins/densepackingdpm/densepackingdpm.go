package densepackingdpm

import (
	"context"
	"fmt"
	"math"
	"time" 

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	framework "k8s.io/kubernetes/pkg/scheduler/framework"
    fwk "k8s.io/kube-scheduler/framework"
)

const Name = "DensePackingDPM"

type DensePackingDPM struct {
	handle framework.Handle
}

var _ framework.ScorePlugin = &DensePackingDPM{}

func New(_ context.Context, _ runtime.Object, h framework.Handle) (framework.Plugin, error) {
    fmt.Println("⚠️ ResourceVectorSimilarity NEW() CALLED!")
	return &DensePackingDPM{handle: h}, nil
}

func (v *DensePackingDPM) Name() string {
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

// Η συνάρτηση getUptimeScoreComponent υπολογίζει ένα βαθμό στο διάστημα (0.0 προς 1.0) βασισμένο στο uptime του κόμβου.
// Όσο περισσότερο είναι ενεργός ένας κόμβος, τόσο μεγαλύτερος ο βαθμός
func getUptimeScoreComponent(node *v1.Node) float64 {

      // Θεωρούμε πως κατά το χειρότερο ένας κόμβος τρέχει για 1 ώρα και όποιος κόμβος είναι ενεργός για 7 μέρες θεωρείται πολύ καλός. Προτείνεται σε μελλοντικές εφαρμογές η τιμή αυτή να είναι μεγαλύτερη
	minUptimeDuration := 1 * time.Hour  
	maxUptimeDuration := 7 * 24 * time.Hour 

      // Βρίσκουμε αν ο κόμβος είναι έτοιμος και από πότε.
	var readyTime time.Time
	foundReady := false
	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady && condition.Status == v1.ConditionTrue {
			readyTime = condition.LastTransitionTime.Time
			foundReady = true
			break
		}
	}
      // Αν δεν βρεθεί ready state δεν θεωρείται καλό για να δρομολογηθεί εκεί ένα Pod.
	if !foundReady {
		return 0.0
	}

      // Υπολογίζουμε το uptime
	uptime := time.Since(readyTime)

	if uptime < minUptimeDuration {
		return 0.0
	}
	if uptime >= maxUptimeDuration {
		return 1.0
	}

	// Βάζουμε γραμμικά το uptime μεταξύ των τιμών 0.0 και 1.0
	return float64(uptime-minUptimeDuration) / float64(maxUptimeDuration-minUptimeDuration)
}


// Η συνάρτηση αξιολόγησης αξιολογεί τους κόμβους βάση της μέγιστης χρήσης CPU και μνήμης αλλά και με βάση το uptime τους.
func (v *DensePackingDPM) Score(
	ctx context.Context,
	state fwk.CycleState,
	pod *v1.Pod,
	nodeInfo *framework.NodeInfo,
) (int64, *framework.Status) {
      // Βάζουμε τις τιμές που χρειαζόμαστε από το κόμβο και το Pod στις μεταβλητές
	allocatableMilliCPU := float64(nodeInfo.Allocatable.MilliCPU)
	allocatableMemory := float64(nodeInfo.Allocatable.Memory)

	requestedMilliCPU := float64(nodeInfo.Requested.MilliCPU)
	requestedMemory := float64(nodeInfo.Requested.Memory)

	podReq := getPodResourceRequest(pod)
	podReqMilliCPU := float64(podReq.MilliCPU)
	podReqMemory := float64(podReq.Memory)

	// Υπολογίζουμε το dense packing με βάση τη χρήση των πόρων.
	predictedUsedMilliCPU := requestedMilliCPU + podReqMilliCPU
	predictedUsedMemory := requestedMemory + podReqMemory

      // Υπολογίζουμε τις προβλεπόμενες χρήσης της CPU και της μνήμης
	predictedCPUUtil := 0.0
	if allocatableMilliCPU > 0 {
		predictedCPUUtil = predictedUsedMilliCPU / allocatableMilliCPU
	}

	predictedMemUtil := 0.0
	if allocatableMemory > 0 {
		predictedMemUtil = predictedUsedMemory / allocatableMemory
	}

	// Η κατά μέσο όρο χρησιμοποίηση CPU και μνήμης (0.0 - 1.0)
	resourceUtilScoreComponent := (predictedCPUUtil + predictedMemUtil) / 2.0


	// Υπολογίζουμε το uptime
	uptimeScoreComponent := getUptimeScoreComponent(nodeInfo.Node())


	// Θέτουμε τα βάρη για τα διαφορετικά Score.
	const resourceWeight = 0.50
	const uptimeWeight = 0.50

	// Ο μέσος όρος των τιμών μετά το πολλαπλασιασμό τους με τα προεπιλεγμένα βάρη. (το καθένα είναι 0.0-1.0)
	combinedScoreRaw := (resourceUtilScoreComponent * resourceWeight) + (uptimeScoreComponent * uptimeWeight)

	// Κανονικοποιούμε το score στη κλίμακα από το 1 ως το 100, με τη μονάδα στο combinesScoreRaw να αντιστοιχεί στο 100 και το 0 στο 1.
	finalScore := int64(1 + combinedScoreRaw*99.0)

	finalScore = int64(math.Max(1, math.Min(100, float64(finalScore))))

	fmt.Printf("DensePackingDPM - Scoring Node %s (Dense Packing + Uptime): Current CPU Util = %.2f%%, Mem Util = %.2f%% | Pred CPU Util = %.2f%%, Mem Util = %.2f%% | Resource Comp = %.2f, Uptime Comp = %.2f, Final Score = %d\n",
		nodeInfo.GetName(),
		(requestedMilliCPU/allocatableMilliCPU)*100, (requestedMemory/allocatableMemory)*100,
		predictedCPUUtil*100, predictedMemUtil*100,
		resourceUtilScoreComponent, uptimeScoreComponent,
		finalScore)

	return finalScore, framework.NewStatus(framework.Success)
}

func (v *DensePackingDPM) ScoreExtensions() framework.ScoreExtensions {
	return nil
}