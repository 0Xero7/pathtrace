package main

type RefractiveIndexTracker struct {
	currentIndex float32
	history      []float32
}

func NewRefractiveIndexTracker(initialIndex float32) *RefractiveIndexTracker {
	return &RefractiveIndexTracker{
		currentIndex: initialIndex,
		history:      []float32{initialIndex},
	}
}

func (r *RefractiveIndexTracker) UpdateIndex(newIndex float32) {
	r.currentIndex = newIndex
	r.history = append(r.history, newIndex)
}

func (r *RefractiveIndexTracker) GetCurrentIndex() float32 {
	if len(r.history) == 0 {
		return 1.0
	}
	return r.history[len(r.history)-1]
}

func (r *RefractiveIndexTracker) GetPreviousIndex() float32 {
	if len(r.history) < 2 {
		return 1.0
	}
	return r.history[len(r.history)-2]
}

func (r *RefractiveIndexTracker) PopIndex() float32 {
	value := r.GetCurrentIndex()
	if len(r.history) > 0 {
		r.history = r.history[:len(r.history)-1]
	}
	return value
}
