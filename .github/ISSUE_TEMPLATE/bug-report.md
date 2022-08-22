---
name: Bug Report
about: Report a bug encountered

---
<!-- Please use this template while reporting a bug and provide as much info as possible. Not doing so may result in your bug not being addressed in a timely manner. Thanks!-->


**What happend**:

**What you expected to happen**:

**How to reproduce it (as minimally and precisely as possible)**:

**Anything else we need to know?**:

**Environment**:

  image path and image ID (from 'docker images')
- Kubernetes version (use `kubectl version`):
- Primary CNI for Kubernetes cluster:
- NetworkAttachment info (use `kubectl get net-attach-def -o yaml`)
- Target pod yaml info (with annotation, use `kubectl get pod <podname> -o yaml`)

