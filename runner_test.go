
// func now() int64 { return time.Now().Unix() }
// func days(count float64) float64 {
// 	return count * 24 * 60 * 60
// }

// func main() {
// 	var sum int64
// 	count := int64(1000000)
// 	funcMap := map[string]interface{}{
// 		"now":  now,
// 		"days": days,
// 	}
// 	inputMap := map[string]interface{}{
// 		"registered_date": time.Now().Unix(),
// 		"sales_amount":    2000000,
// 	}
// 	outputMap := map[string]interface{}{
// 		"plan_name": "premium_1",
// 	}
// 	t := time.Now().UnixNano()
// 	a, err := parse.Parse([]string{
// 		`now() > registered_date + days(14) && plan_name == "premium_1" | plan_name = "free"`,
// 		`plan_name == "premium_1" | plan_name = "free"`,
// 		`plan_name == "free" | feature_1 = true`,
// 	}, funcMap, inputMap, outputMap)
// 	x := time.Duration(time.Now().UnixNano() - t)
// 	t = time.Now().UnixNano()
// 	fmt.Println(x.Nanoseconds())
// 	fmt.Sprint(err)
// 	e, err := eval.New(funcMap, inputMap, outputMap)
// 	for i := int64(0); i < count; i++ {
// 		t := time.Now().UnixNano()
// 		r, err := e.Eval(a)
// 		x := time.Duration(time.Now().UnixNano() - t)
// 		t = time.Now().UnixNano()
// 		plan_name := "premium_1"
// 		registered_date := time.Now().Unix()
// 		sales_amount := 2000000
// 		if float64(now()) > float64(registered_date)+days(14) && plan_name == "premium_1" {
// 			plan_name = "free"
// 		}
// 		if plan_name == "premium_1" {
// 			plan_name = "free"
// 		}
// 		var feature_1 bool
// 		if plan_name == "free" {
// 			feature_1 = true
// 		}
// 		x1 := time.Duration(time.Now().UnixNano() - t)
// 		fmt.Sprint(feature_1, sales_amount)
// 		sum = sum + (x.Nanoseconds() / x1.Nanoseconds())
// 		if err != nil {
// 			fmt.Println(err)
// 			return
// 		}
// 		fmt.Sprint(r)
// 	}
// 	fmt.Println(sum / count)
// }
