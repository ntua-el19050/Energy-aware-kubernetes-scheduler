package energyawareeatsvm

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

const Name = "EnergyAwareEATSVM"

const nodeClockSpeed = 2.67

type EnergyAwareEATSVM struct {
	handle framework.Handle
}

var _ framework.ScorePlugin = &EnergyAwareEATSVM{}

func New(_ context.Context, _ runtime.Object, h framework.Handle) (framework.Plugin, error) {
    fmt.Println("⚠️ EnergyAwareEATSVM NEW() CALLED!")
	return &EnergyAwareEATSVM{handle: h}, nil
}

func (v *EnergyAwareEATSVM) Name() string {
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

// Αντίστοιχη συνάρτηση αξιολόγησης. Παίρνουμε δεδομένα χρήσης από τον κόμβο και το Pod στην αρχή και στη συνέχεια βλέπουμε τη προβλεπόμενη χρήση της CPU με την προσθήκη του Pod και υπολογίζουμε την ισχύ που θα καταναλώνει. Υπολογίζουμε το τελικό score και όσο μεγαλύτερο είναι, τόσο το καλύτερο.
func (v *EnergyAwareEATSVM) Score(
	ctx context.Context,
	state fwk.CycleState,
	pod *v1.Pod,
	nodeInfo *framework.NodeInfo,
) (int64, *framework.Status) {

      // Έχουμε σε μεταβλητές την μέγιστη πιθανή CPU, την χρήση CPU από τον κόμβο αλλά και τη χρήση που ζητάει το pod.
	allocatableMilliCPU := float64(nodeInfo.Allocatable.MilliCPU)
	requestedMilliCPU := float64(nodeInfo.Requested.MilliCPU)

	podReq := getPodResourceRequest(pod)
	podReqMilliCPU := float64(podReq.MilliCPU)

      // Η προβλεπόμενη χρήση CPU είναι η CPU που χρησιμοποιείται ήδη μαζί με τη CPU που ζητείται από το Pod, και μετά τη βρίσκουμε ως ποσοστό προς τη συνολική CPU που μπορεί να δοθεί
	predictedUsedMilliCPU := requestedMilliCPU + podReqMilliCPU
	predictedCPUUtil := 0.0
	if allocatableMilliCPU > 0 {
		predictedCPUUtil = predictedUsedMilliCPU / allocatableMilliCPU
	}

	// Παίρνουμε τη τιμή της ταχύτητας του ρολογιού από τα annotations
    clockSpeed := nodeClockSpeed
    if val, ok := nodeInfo.Node().Annotations["clockSpeed"]; ok {
        if parsed, err := strconv.ParseFloat(val, 64); err == nil {
            clockSpeed = parsed
        }
    }

      // Η συνολική προβλεπόμενη ισχύ υπολογίζεται ως το ποσοστό χρήσης της CPU επί τους κύκλους του ρολογιού, που μπορεί να είναι από 0 έως και τους κύκλους του ρολογιού
	predictedTotalPower := clockSpeed * predictedCPUUtil

	minExpectedPower := 0.0
	maxExpectedPower := nodeClockSpeed * 1.0 
	clampedPower := math.Max(minExpectedPower, math.Min(maxExpectedPower, predictedTotalPower))

      // Κανονικοποιούμε την τιμή μεταξύ του 0 και του 1 πριν την επιστρέψουμε στον χρονοδρομολογητή.
	normalizedValue := 0.0
	if maxExpectedPower > minExpectedPower {
		normalizedValue = (clampedPower - minExpectedPower) / (maxExpectedPower - minExpectedPower)
	}

      // Κανονικοποιούμε την τιμή μεταξύ του 1 και του 100, αλλά επειδή θέλουμε η δρομολόγηση να ευνοεί τους κόμβους με καλύτερη λιγότερη παραγωγή ισχύος, το αντιστρέφουμε.
	finalScore := int64(1 + (1.0-normalizedValue)*99.0)
	finalScore = int64(math.Max(1, math.Min(100, float64(finalScore))))

	fmt.Printf("EnergyAwareEATSVM - Scoring Node %s: Clock Speed = %.2f, Predicted CPU Util = %.2f%%, Predicted Total Power = %.4f, Score = %d\n",
		nodeInfo.GetName(), clockSpeed, predictedCPUUtil*100, predictedTotalPower, finalScore)

	return finalScore, framework.NewStatus(framework.Success)
}

func (v *EnergyAwareEATSVM) ScoreExtensions() framework.ScoreExtensions {
	return nil
}