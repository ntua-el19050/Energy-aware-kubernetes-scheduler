package resourcevectorsimilarity

import (
	"context"
	"fmt"
	"math"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	framework "k8s.io/kubernetes/pkg/scheduler/framework"
    fwk "k8s.io/kube-scheduler/framework"
)

const Name = "ResourceVectorSimilarity"

type ResourceVectorSimilarity struct {
	handle framework.Handle
}

var _ framework.ScorePlugin = &ResourceVectorSimilarity{}

func New(_ context.Context, _ runtime.Object, h framework.Handle) (framework.Plugin, error) {
            fmt.Println("⚠️ ResourceVectorSimilarity NEW() CALLED!")
	return &ResourceVectorSimilarity{handle: h}, nil
}

func (v *ResourceVectorSimilarity) Name() string {
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

// Σε αυτή τη συνάρτηση αξιολόγησης φτιάχνουμε 2 διανύσματα, το ένα με βάση την CPU και τη μνήμη που απομένει στον κόμβο και το άλλο με βάση την CPU και τη μνήμη που ζητάει το Pod.
func (v *ResourceVectorSimilarity) Score(
	ctx context.Context,
	state fwk.CycleState,
	pod *v1.Pod,
	nodeInfo *framework.NodeInfo,
) (int64, *framework.Status) {

      // Βάζουμε στις μεταβλητές τις τιμές που έχουμε από τον κόμβο και το Pod σχετικά με τη CPU και τη μνήμη.
	allocatableMilliCPU := float64(nodeInfo.Allocatable.MilliCPU)
	allocatableMemory := float64(nodeInfo.Allocatable.Memory)

	requestedMilliCPU := float64(nodeInfo.Requested.MilliCPU)
	requestedMemory := float64(nodeInfo.Requested.Memory)

	podReq := getPodResourceRequest(pod)
	podReqMilliCPU := float64(podReq.MilliCPU)
	podReqMemory := float64(podReq.Memory)

      // Υπολογίζουμε τη CPU και τη μνήμη που απομένει στο κόμβο 
	//cpuRemaining := allocatableMilliCPU - requestedMilliCPU
	//memRemaining := allocatableMemory - requestedMemory

      // Βάζουμε τις τιμές που θέλουμε στα διανύσματα μας
	vectorA_CPU := requestedMilliCPU
	vectorA_Mem := requestedMemory/(1024.0*1024.0)

	vectorB_CPU := podReqMilliCPU
	vectorB_Mem := podReqMemory/(1024.0*1024.0)

	// Υπολογίζουμε το Dot Product
	dotProduct := (vectorA_CPU * vectorB_CPU) + (vectorA_Mem * vectorB_Mem)

	// Υπολογίζουμε το Cosine Similarity
	magnitudeA := math.Sqrt(vectorA_CPU*vectorA_CPU + vectorA_Mem*vectorA_Mem)
	magnitudeB := math.Sqrt(vectorB_CPU*vectorB_CPU + vectorB_Mem*vectorB_Mem)

	cosineSimilarity := 0.0
	if magnitudeA > 0 && magnitudeB > 0 {
		cosineSimilarity = dotProduct / (magnitudeA * magnitudeB)
	}

    // minexpecteddotproduct := 0.0
    maxExpectedDotProduct := (allocatableMilliCPU * podReqMilliCPU) + ((allocatableMemory/(1024.0*1024.0)) * (podReqMemory/(1024.0*1024.0)))

    normalizedDotProduct := math.Min(1.0, dotProduct / maxExpectedDotProduct) 
    normalizedDotProduct = math.Max(0.0, normalizedDotProduct)

    // Υπολογίζουμε το τελικό αποτέλεσμα με ποσοστό 3/4 για το dot product και 1/4 για το Cosine similarity	
	finalScore := int64(1 + normalizedDotProduct*75 + cosineSimilarity*25) 

	finalScore = int64(math.Max(1, math.Min(100, float64(finalScore))))

	fmt.Printf("ResourceVectorSimilarity - Scoring Node %s: Rem CPU = %dm, Rem Mem = %dMi | Pod Req CPU = %dm, Pod Req Mem = %dMi | Dot Product = %.2f, MagnitudeA = %.2f, MagnitudeB = %.2f, Cosine Similarity = %.4f, Score = %d\n",
		nodeInfo.GetName(), int64(vectorA_CPU), int64(vectorA_Mem)/(1024*1024),
		int64(vectorB_CPU), int64(vectorB_Mem)/(1024*1024),
		normalizedDotProduct, magnitudeA, magnitudeB, cosineSimilarity, finalScore)

	return finalScore, framework.NewStatus(framework.Success)
}

func (v *ResourceVectorSimilarity) ScoreExtensions() framework.ScoreExtensions {
	return nil
}