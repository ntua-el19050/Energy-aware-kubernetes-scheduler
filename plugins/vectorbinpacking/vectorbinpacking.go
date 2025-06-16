package vectorbinpacking

// Όλα τα imports γίνονται εδώ. Τα περισσότερα χρειάζονται για την επικοινωνία με το Kubernetes. Υπάρχουν 2 frameworks για το kube-scheduler καθώς πλέον το CycleState παρέχεται από το k8s.io/kube-scheduler/framework
import ( 
    "context"
    "fmt"
    "math"

    v1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/runtime"
    framework "k8s.io/kubernetes/pkg/scheduler/framework"
    fwk "k8s.io/kube-scheduler/framework"
)

const Name = "VectorBinPacking"

// Δημιουργούμε το Struct για να μπορούμε να το καλέσουμε αργότερα
type VectorBinPacking struct {
	handle framework.Handle
}

var _ framework.ScorePlugin = &VectorBinPacking{}

// Η συνάρτηση New δημιουργεί το instance του plugin μας στον kube-scheduler. Το μήνυμα στο logger υπάρχει ώστε να είμαστε σίγουρη για την ορθή εκκίνηση του plugin.
func New(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
    fmt.Println("⚠️ VectorBinPacking NEW() CALLED!")
    return &VectorBinPacking{}, nil
}

// Επιστρέφει το όνομα του plugin
func (v *VectorBinPacking) Name() string {
    return Name
}

// Χρησιμοποιούμε αυτή τη συνάρτηση για να πάρουμε τα Pod Requested Resources καθώς η συνάρτηση που ήταν υλοποιημένη δεν λειτουργεί, οπότε την καλούμε για να γνωρίζουμε πως γίνεται με σωστό τρόπο.
func getPodResourceRequest(pod *v1.Pod) *framework.Resource {
    result := &framework.Resource{}
    // Υπολογίζετια η συνολική CPU που ζητείται από τα containers αλλά και η συνολική μνήμη που ζητείται
    for _, c := range pod.Spec.Containers {
        result.MilliCPU += c.Resources.Requests.Cpu().MilliValue()
        result.Memory += c.Resources.Requests.Memory().Value()
    }
    // Στη περίπτωση που υπάρχει InitContainer υποχρεωτικά ζητείται τουλάχιστον η CPU και η μνήμη που απαιτεί αυτός. Αν τα υπόλοιπα containers απαιτούν λιγότερη, η δική του υπερισχύει διότι διαφορετικά δεν μπορεί να λειτουργήσει το Pod.
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

// Η συνάρτηση αξιολόγησης. Για τα επόμενα plugins κυρίως αυτή θα παρέχεται. Πρώτα ορίζουμε τις μεταβλητές και στη συνέχεια αρχικοποιούμε το σύστημα μας βάσει των πληροφοριών περί CPU και μνήμη κόμβου και Pod. Το τελικό αποτέλεσμα δείχνει κατά πόσο το μήκος του vector είναι κοντά στο 0. Όσο πιο κοντά στο 0 είναι, τόσο περισσότερο το προτιμάμε.  
func (v *VectorBinPacking) Score(
    ctx context.Context,
    state fwk.CycleState,
    pod *v1.Pod,
    nodeInfo *framework.NodeInfo,
) (int64, *framework.Status) {
    // Πρώτα υπολογίζουμε πόση CPU και μνήμη μπορεί να αξιοποιηθεί συνολικά, και πόση αξιοποιείται ήδη.
    allocatable := nodeInfo.Allocatable
    requested := nodeInfo.Requested

    // Στη συνέχεια υπολογίζουμε πόση CPU και μνήμη απαιτείται από το Pod που δρομολογείται.
    podReq := getPodResourceRequest(pod)

    // Υπολογίζεται η CPU και η μνήμη που όντως απομένει στον κόμβο για να παρέχει.
    cpuRemaining := allocatable.MilliCPU - requested.MilliCPU
    memRemaining := allocatable.Memory - requested.Memory

    // Υπολογίζεται η CPU και η μνήμη που θα απομένει στον κόμβο μετά τη προσθήκη του Pod.
    cpuLeft := cpuRemaining - podReq.MilliCPU
    memLeft := memRemaining - podReq.Memory

    if cpuLeft < 0 || memLeft < 0 {
        return 0, framework.NewStatus(framework.Success)
    }

    // Vector bin packing: όσο πιο κοντά στο 0 είναι το διάνυσμα, τόσο το καλύτερο. Η διαίρεση γίνεται με την υψηλότερη δυνατή τιμή από το άθροισμα των τετραγώνων για την συνολική CPU και τη συνολική μνήμη που μπορεί να δοθεί.
    cpuLeftF := float64(cpuLeft)
    memLeftF := float64(memLeft) / (1024.0 * 1024.0)
    allocCPUF := float64(allocatable.MilliCPU)
    allocMemF := float64(allocatable.Memory) /(1024.0*1024.0) // Μετατροπή από bytes σε MiB

    denominator := math.Sqrt(allocCPUF*cpuLeftF + allocMemF*memLeftF)
    numerator := math.Sqrt(cpuLeftF*cpuLeftF + memLeftF*memLeftF)
    
    score := int64(math.Round(100.0 - (numerator / denominator)*100.0))
    score = int64(math.Max(1, math.Min(100, float64(score))))
    
    // Παρουσιάζουμε τα αποτελέσματα στο Log για τη δυνατότητα ελέγχου του plugin.
    fmt.Printf("Scoring Node %s: CPU Allocatable = %dm, Mem Allocatable = %dMi, CPU left = %dm, Mem left = %dMi, Score = %d\n", nodeInfo.GetName(), allocatable.MilliCPU, allocatable.Memory/(1024*1024), cpuLeft, memLeft/(1024*1024), score)
    return score, framework.NewStatus(framework.Success)
}

// Συνάρτηση που χρειάζεται για SocreExtensions αλλά δεν απαιτεί παραπάνω κώδικα. Δεν θα παρουσιαστεί για τα επόμενα plugins. 
func (v *VectorBinPacking) ScoreExtensions() framework.ScoreExtensions {
    return nil
}

