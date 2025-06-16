# soumplis-piliotis-1
microk8s kubectl annotate node node-1 greenFactor="1.8" --overwrite
microk8s kubectl annotate node node-1 carbonPenalty="1.2" --overwrite
microk8s kubectl annotate node node-1 clockSpeed="3.27" --overwrite

# soumplis-piliotis-2
microk8s kubectl annotate node node-2 greenFactor="1.6" --overwrite
microk8s kubectl annotate node node-2 carbonPenalty="0.8" --overwrite
microk8s kubectl annotate node node-2 clockSpeed="2.67" --overwrite

# soumplis-piliotis-3
microk8s kubectl annotate node node-3 greenFactor="0.8" --overwrite
microk8s kubectl annotate node node-3 carbonPenalty="1.5" --overwrite
microk8s kubectl annotate node node-3 clockSpeed="1.89" --overwrite

