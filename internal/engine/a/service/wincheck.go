package service

import "math"

func winSatisfied(metrics map[string]float64, checks []WinCheck) bool {
	for _, c := range checks {
		v, ok := metrics[c.Metric]
		if !ok {
			return false
		}
		switch c.Op {
		case ">":
			if !(v > c.Value) {
				return false
			}
		case "<":
			if !(v < c.Value) {
				return false
			}
		case ">=":
			if !(v >= c.Value) {
				return false
			}
		case "<=":
			if !(v <= c.Value) {
				return false
			}
		case "==":
			if math.Abs(v-c.Value) > 1e-9 {
				return false
			}
		default:
			return false
		}
	}
	return true
}
