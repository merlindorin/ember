package main

import "github.com/lantervale/ember/ease"

// easings are the families shown, a row each, through their InOut form:
// each family also has an In form, slow at the start, and an Out form,
// slow at the end.
// easing is a family's name and its InOut form.
type easing struct {
	name string
	f    ease.Func
}

func easings() []easing {
	return []easing{
		{"Linear", ease.Linear},
		{"Quad", ease.InOutQuad},
		{"Cubic", ease.InOutCubic},
		{"Sine", ease.InOutSine},
		{"Expo", ease.InOutExpo},
		{"Back", ease.InOutBack},
		{"Elastic", ease.InOutElastic},
		{"Bounce", ease.InOutBounce},
	}
}
