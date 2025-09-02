package main

type RefractiveIndexTracker struct {
	currentIndex float64
	history      []float64
}

func NewRefractiveIndexTracker(initialIndex float64) *RefractiveIndexTracker {
	return &RefractiveIndexTracker{
		currentIndex: initialIndex,
		history:      []float64{initialIndex},
	}
}

func (r *RefractiveIndexTracker) UpdateIndex(newIndex float64) {
	r.currentIndex = newIndex
	r.history = append(r.history, newIndex)
}

func (r *RefractiveIndexTracker) GetCurrentIndex() float64 {
	if len(r.history) == 0 {
		return 1.0
	}
	return r.history[len(r.history)-1]
}

func (r *RefractiveIndexTracker) GetPreviousIndex() float64 {
	if len(r.history) < 2 {
		return 1.0
	}
	return r.history[len(r.history)-2]
}

func (r *RefractiveIndexTracker) PopIndex() float64 {
	value := r.GetCurrentIndex()
	if len(r.history) > 0 {
		r.history = r.history[:len(r.history)-1]
	}
	return value
}
