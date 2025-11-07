package linear

import "math"

type Options struct {
	Epochs       int
	LearningRate float64
}

type Weights struct {
	Bias         float64   `json:"bias"`
	Coefficients []float64 `json:"coefficients"`
}

type Metrics struct {
	Loss     float64
	Accuracy float64
}

func TrainLogistic(samples [][]float64, labels []float64, opts Options) (Weights, Metrics) {
	if opts.Epochs <= 0 {
		opts.Epochs = 200
	}
	if opts.LearningRate <= 0 {
		opts.LearningRate = 0.01
	}

	n := len(samples)
	if n == 0 {
		return Weights{}, Metrics{}
	}
	featureCount := len(samples[0])
	weights := make([]float64, featureCount)
	var bias float64

	for epoch := 0; epoch < opts.Epochs; epoch++ {
		grad := make([]float64, featureCount)
		var biasGrad float64
		for i, sample := range samples {
			prediction := sigmoid(dot(weights, sample) + bias)
			error := prediction - labels[i]
			for j := 0; j < featureCount; j++ {
				grad[j] += error * sample[j]
			}
			biasGrad += error
		}
		for j := 0; j < featureCount; j++ {
			weights[j] -= opts.LearningRate * grad[j] / float64(n)
		}
		bias -= opts.LearningRate * biasGrad / float64(n)
	}

	loss, accuracy := evaluate(weights, bias, samples, labels)
	return Weights{Bias: bias, Coefficients: weights}, Metrics{Loss: loss, Accuracy: accuracy}
}

func Predict(weights Weights, sample []float64) float64 {
	return sigmoid(dot(weights.Coefficients, sample) + weights.Bias)
}

func dot(weights []float64, sample []float64) float64 {
	var sum float64
	for i := 0; i < len(weights); i++ {
		sum += weights[i] * sample[i]
	}
	return sum
}

func sigmoid(x float64) float64 {
	return 1 / (1 + math.Exp(-x))
}

func evaluate(weights []float64, bias float64, samples [][]float64, labels []float64) (float64, float64) {
	var loss float64
	var correct int
	for i, sample := range samples {
		prediction := sigmoid(dot(weights, sample) + bias)
		loss += -labels[i]*math.Log(prediction+1e-9) - (1-labels[i])*math.Log(1-prediction+1e-9)
		if (prediction >= 0.5 && labels[i] == 1) || (prediction < 0.5 && labels[i] == 0) {
			correct++
		}
	}
	loss /= float64(len(samples))
	accuracy := float64(correct) / float64(len(samples))
	return loss, accuracy
}
