Example:

```go
 func now() int64 { return time.Now().Unix() }
 func days(count float64) float64 {
 	return count * 24 * 60 * 60
 }

 func main() {
 	funcMap := map[string]interface{}{
		"now":  now,
		"days": days,
	}
	inputMap := map[string]interface{}{
		"registered_date": time.Now().Unix(),
		"sales_amount":    2000000,
	}
	outputMap := map[string]interface{}{
		"plan_name": "premium_1",
	}
	output, err := mosalat.Run([]string{
		`now() > registered_date + days(14) && plan_name == "premium_1" | plan_name = "free"`,
		`plan_name == "premium_1" | plan_name = "free"`,
		`plan_name == "free" | feature_1 = true`,
	}, funcMap, inputMap, outputMap) // --> [plan_name: "free", feature_1: true]
 }
```
