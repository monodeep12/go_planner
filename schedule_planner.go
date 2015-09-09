package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var input_data map[string]map[string][]string
var geo_list []string
var max_spot_length = flag.Int("s", 0, "max spot length")
var average_cost_per_10_sec = 13000
var skip = flag.Int("skip", 5, "skip")
var sweep_duration_list = []int{}
var best_combination_by_geo = [][]int{}
var best_duration_by_geo = []int{}
var best_revenue_by_geo = []int{}
var wg sync.WaitGroup
var before_pruning = 0
var after_pruning = 0
var sum_branch_num = 0
var branch_func_counter = 0
var maxbranch = 0

func flatten_rotates(rotates []string, max_rotates []string, back_to_back []string, back_to_back_max_rotates []string,
	back_to_back_min_duration []string, duration []string) [][]int {

	flattened_rotates := [][]int{}
	for i, _ := range rotates {
		temp_arr := []int{}
		max_rotate, _ := strconv.Atoi(max_rotates[i])

		for j := 0; j <= max_rotate; j++ {
			b_to_b, _ := strconv.Atoi(back_to_back[i])
			b_to_b_m_r, _ := strconv.Atoi(back_to_back_max_rotates[i])
			b_to_b_m_d, _ := strconv.Atoi(back_to_back_min_duration[i])
			if j > 1 && b_to_b == 1 {
				if j <= b_to_b_m_r {
					temp_arr = append(temp_arr, j)
				}
			} else if j > 1 && b_to_b == 0 {
				if j <= b_to_b_m_r {
					if b_to_b_m_d != 0 {
						min_duration := b_to_b_m_d
						min_captions_required := j - 1
						count := 0
						for k, _ := range rotates {
							dur_of_k, _ := strconv.Atoi(duration[k])
							dur_of_i, _ := strconv.Atoi(duration[i])
							b_to_b_of_k, _ := strconv.Atoi(back_to_back[k])
							b_to_b_m_d_of_k, _ := strconv.Atoi(back_to_back_min_duration[k])
							r_of_k, _ := strconv.Atoi(rotates[k])
							if k != i {
								if dur_of_k >= min_duration {
									if b_to_b_of_k == 1 || (b_to_b_m_d_of_k != 0 && b_to_b_m_d_of_k <= dur_of_i) {
										count = count + r_of_k
										if count >= min_captions_required {
											temp_arr = append(temp_arr, j)
											break
										} else if b_to_b_of_k == 0 && b_to_b_m_d_of_k == 0 {
											count += 1
											if count >= min_captions_required {
												temp_arr = append(temp_arr, j)
												break
											}
										}
									}
								}
							}
						}
					}
				}
			} else {
				temp_arr = append(temp_arr, j)
			}
		}
		flattened_rotates = append(flattened_rotates, temp_arr)
	}
	return flattened_rotates
}

// This function is a python implementation of itertools.product
func cartesian_product(duration []string, args ...[]int) [][]int {

	pools := args
	npools := len(pools)
	indices := make([]int, npools)

	result := make([]int, npools)
	/*
		// This loop generates the first product(combination)
		// Commented out because this  is  always 0 in our use case
		for i := range result {
			if len(pools[i]) == 0 {
				return nil
			}
			result[i] = pools[i][0]
		}
		results := [][]int{result}*/

	results := [][]int{}

	for {
		i := npools - 1
		for ; i >= 0; i -= 1 {
			pool := pools[i]
			indices[i] += 1

			if indices[i] == len(pool) {
				indices[i] = 0
				result[i] = pool[0]
			} else {
				result[i] = pool[indices[i]]
				break
			}

		}

		if i < 0 {
			return results
		}

		temp_duration := 0
		for idx, r := range result {
			d, _ := strconv.Atoi(duration[idx])
			temp_duration += r * d
		}

		if temp_duration <= *max_spot_length {
			newresult := make([]int, npools)
			copy(newresult, result)
			//fmt.Println(newresult)
			results = append(results, newresult)
		}

	}

	return nil
}

func apply_multiple_caption_combination_constraint(flattened_combinations_pruned [][]int, captions []string,
	multiple_caption_combination []string) [][]int {
	caption_dict := map[string]int{}
	temp_slice := [][]int{}
	final_slice := [][]int{}
	for idx, c := range captions {
		caption_dict[c] = idx
	}
	for _, comb := range flattened_combinations_pruned {
		for idx_c, _ := range captions {
			if multiple_caption_combination[idx_c] != "" {
				captions_not_allowed := strings.Split(multiple_caption_combination[idx_c], "|")
				if comb[idx_c] != 0 {
					for _, j := range captions_not_allowed {
						caption_index := caption_dict[j]
						if comb[caption_index] != 0 {
							temp_slice = append(temp_slice, comb)
							break
						}
					}

				}
			}
		}
	}

	for _, comb := range flattened_combinations_pruned {
		if sliceInSlice(comb, temp_slice) == false {
			final_slice = append(final_slice, comb)
		}
	}
	return final_slice
}

func get_combinations_less_than_duration(final_combinations [][]int, duration []string, effective_rate []string, target int) ([][]int, []int, []int) {
	filtered_combinations := [][]int{}
	filtered_durations := []int{}
	filtered_revenue := []int{}

	for _, comb := range final_combinations {
		temp_duration := 0
		temp_revenue := 0
		for idx, _ := range comb {
			d, _ := strconv.Atoi(duration[idx])
			e, _ := strconv.Atoi(effective_rate[idx])
			temp_duration += comb[idx] * d
			spot_effective_rate := d / 10 * e
			temp_revenue += comb[idx] * spot_effective_rate
		}

		if temp_duration <= target {
			filtered_combinations = append(filtered_combinations, comb)
			filtered_durations = append(filtered_durations, temp_duration)
			filtered_revenue = append(filtered_revenue, temp_revenue)
		}
	}
	return filtered_combinations, filtered_durations, filtered_revenue
}

func get_backup_combination_for_geo(filtered_combinations [][]int, filtered_durations []int, filtered_revenue []int,
	min_rotates_lst []string, geo string) ([][]int, []int, []int) {
	if len(filtered_combinations) == 0 {
		for i := 0; i <= len(min_rotates_lst); i++ {
			temp_lst := []int{0}
			best_combination_by_geo = append(best_combination_by_geo, temp_lst)
		}

		best_duration_by_geo = append(best_duration_by_geo, 0)
		best_revenue_by_geo = append(best_revenue_by_geo, 0)

		return best_combination_by_geo, best_duration_by_geo, best_revenue_by_geo
	} else {
		weight := map[string]int{"margin": 1, "min": 100000}
		min_rotates := []int{}
		if len(min_rotates_lst) == 0 {
			weight["min"] = 0
			for i := 0; i <= len(filtered_combinations[0]); i++ {
				min_rotates = append(min_rotates, 0)
			}
		} else {
			for _, k := range min_rotates_lst {
				x, _ := strconv.Atoi(k)
				if x <= 0 {
					min_rotates = append(min_rotates, 0)
				} else {
					min_rotates = append(min_rotates, 1)
				}
			}
		}

		min_duration_profit_slice := []int{}
		weighted_profit_slice := []int{}

		for comb_idx, comb := range filtered_combinations {
			min_duration_profit := 0
			for idx, c := range comb {
				d, _ := strconv.Atoi(input_data[geo]["duration"][idx])
				min_duration_profit += min_rotates[idx] * d * c
			}
			min_duration_profit_slice = append(min_duration_profit_slice, min_duration_profit)
			weighted_profit_slice = append(weighted_profit_slice, filtered_revenue[comb_idx]*weight["margin"]+min_duration_profit*weight["min"])
		}
		best_combination_index, _ := max_in_slice_int(weighted_profit_slice)
		best_combination := filtered_combinations[best_combination_index]
		best_combination_by_geo = append(best_combination_by_geo, best_combination)
		best_duration_by_geo = append(best_duration_by_geo, filtered_durations[best_combination_index])
		best_revenue_by_geo = append(best_revenue_by_geo, filtered_revenue[best_combination_index])
		return best_combination_by_geo, best_duration_by_geo, best_revenue_by_geo
	}
}

func max_in_slice_int(a []int) (int, int) {
	max := a[0]
	index := 0
	for i, v := range a {
		if v > max {
			max = v
			index = i
		}
	}
	return index, max
}

func min_in_slice_int(a []int) (int, int) {
	min := a[0]
	index := 0
	for i, v := range a {
		if v < min {
			min = v
			index = i
		}
	}
	return index, min
}

func sliceInSlice(a interface{}, list interface{}) bool {
	s := reflect.ValueOf(list)
	for i := 0; i < s.Len(); i++ {
		if reflect.DeepEqual(a, s.Index(i).Interface()) {
			return true
		}
	}

	return false
}

func sum_elements_in_slice(a []int) int {
	sum := 0
	for _, v := range a {
		sum += v
	}
	return sum
}

func count_elements_in_slice(a []int, b int) int {
	count := 0
	for _, v := range a {
		if v == b {
			count += 1
		}
	}
	return count
}

func schedule_planner(check bool, combination_tuple_key [][][]int, combination_tuple_value []int, args map[string][][]string) {
	// clearing slices
	best_combination_by_geo = best_combination_by_geo[:0]
	best_duration_by_geo = best_duration_by_geo[:0]
	best_revenue_by_geo = best_revenue_by_geo[:0]

	sweeps_10_to_60_combinations := make([][][]int, len(sweep_duration_list))
	sweeps_10_to_60_durations := make([][]int, len(sweep_duration_list))
	sweeps_10_to_60_revenue := make([][]int, len(sweep_duration_list))
	sweeps_10_to_60_margin := make([][]int, len(sweep_duration_list))

	var sweeps_10_to_60_combinations_chan chan [][][]int = make(chan [][][]int)
	var sweeps_10_to_60_durations_chan chan [][]int = make(chan [][]int)
	var sweeps_10_to_60_revenue_chan chan [][]int = make(chan [][]int)

	// duration_list := [][]string{}
	// rotates_list := [][]string{}
	// captions_list := [][]string{}
	// min_rotates_list := [][]string{}
	// max_rotates_list := [][]string{}
	// back_to_back_list := [][]string{}
	// back_to_back_max_rotates_list := [][]string{}
	// multiple_caption_combination_list := [][]string{}
	// effective_rate_list := [][]string{}
	// back_to_back_min_duration_list := [][]string{}

	duration := []string{}
	rotates := []string{}
	captions := []string{}
	min_rotates := []string{}
	max_rotates := []string{}
	back_to_back := []string{}
	back_to_back_max_rotates := []string{}
	multiple_caption_combination := []string{}
	effective_rate := []string{}
	back_to_back_min_duration := []string{}

	for idx, geo := range geo_list {
		if check {
			fmt.Println("Key:", geo, "Value:", input_data[geo]["caption_names"])
			duration = input_data[geo]["duration"]
			rotates = input_data[geo]["rotates"]
			captions = input_data[geo]["captions"]
			min_rotates = input_data[geo]["min_rotates"]
			max_rotates = input_data[geo]["max_rotates"]
			back_to_back = input_data[geo]["back_to_back"]
			back_to_back_max_rotates = input_data[geo]["back_to_back_max_rotates"]
			multiple_caption_combination = input_data[geo]["multiple_caption_combination"]
			effective_rate = input_data[geo]["effective_rate"]
			back_to_back_min_duration = input_data[geo]["back_to_back_min_duration"]

			args["duration_list"] = append(args["duration_list"], duration)
			args["rotates_list"] = append(args["rotates_list"], rotates)
			args["captions_list"] = append(args["captions_list"], rotates)
			args["min_rotates_list"] = append(args["min_rotates_list"], rotates)
			args["max_rotates_list"] = append(args["max_rotates_list"], rotates)
			args["back_to_back_list"] = append(args["back_to_back_list"], rotates)
			args["back_to_back_max_rotates_list"] = append(args["back_to_back_max_rotates_list"], rotates)
			args["multiple_caption_combination_list"] = append(args["multiple_caption_combination_list"], rotates)
			args["effective_rate_list"] = append(args["effective_rate_list"], rotates)
			args["back_to_back_min_duration_list"] = append(args["back_to_back_min_duration_list"], rotates)
		} else {
			duration = args["duration_list"][idx]
			rotates = args["rotates_list"][idx]
			captions = args["captions_list"][idx]
			min_rotates = args["min_rotates_list"][idx]
			max_rotates = args["max_rotates_list"][idx]
			back_to_back = args["back_to_back_list"][idx]
			back_to_back_max_rotates = args["back_to_back_max_rotates_list"][idx]
			multiple_caption_combination = args["multiple_caption_combination_list"][idx]
			effective_rate = args["effective_rate_list"][idx]
			back_to_back_min_duration = args["back_to_back_min_duration_list"][idx]
		}

		// Step 1: Find all the possible ways a cation can be played
		flattened_rotates := flatten_rotates(rotates, max_rotates, back_to_back, back_to_back_max_rotates, back_to_back_min_duration, duration)
		flattened_combinations_pruned := cartesian_product(duration, flattened_rotates...)
		final_combinations := apply_multiple_caption_combination_constraint(flattened_combinations_pruned, captions, multiple_caption_combination)

		wg.Add(1)
		go generate_sweeps(sweeps_10_to_60_combinations_chan, sweeps_10_to_60_durations_chan, sweeps_10_to_60_revenue_chan, final_combinations, duration, effective_rate, min_rotates, geo, sweeps_10_to_60_combinations, sweeps_10_to_60_durations, sweeps_10_to_60_revenue)
		sweeps_10_to_60_combinations = <-sweeps_10_to_60_combinations_chan
		sweeps_10_to_60_durations = <-sweeps_10_to_60_durations_chan
		sweeps_10_to_60_revenue = <-sweeps_10_to_60_revenue_chan
	}
	fmt.Println("Test--> ", sweeps_10_to_60_combinations)
	fmt.Println("Test--> ", sweeps_10_to_60_durations)
	fmt.Println("Test--> ", sweeps_10_to_60_revenue)

	if check {
		fmt.Println("Test--> ", sweeps_10_to_60_revenue)
		create_tree(sweeps_10_to_60_combinations, sweeps_10_to_60_durations, sweeps_10_to_60_revenue, sweeps_10_to_60_margin, combination_tuple_key, combination_tuple_value, args)
	} else {

	}
}

func generate_sweeps(c chan [][][]int, d chan [][]int, e chan [][]int, final_combinations [][]int, duration []string, effective_rate []string, min_rotates []string, geo string, sweeps_10_to_60_combinations [][][]int, sweeps_10_to_60_durations [][]int, sweeps_10_to_60_revenue [][]int) {
	defer wg.Done()

	for idx_s, s := range sweep_duration_list {
		// Increment the WaitGroup counter.
		wg.Add(1)
		go func(idx_s int, s int) {
			defer wg.Done()

			// clearing slices
			best_combination_by_geo = best_combination_by_geo[:0]
			best_duration_by_geo = best_duration_by_geo[:0]
			best_revenue_by_geo = best_revenue_by_geo[:0]

			filtered_combinations, filtered_durations, filtered_revenue := get_combinations_less_than_duration(final_combinations, duration, effective_rate, s)
			r_combinations, r_durations, r_revenue := get_backup_combination_for_geo(filtered_combinations, filtered_durations, filtered_revenue, min_rotates, geo)
			sweeps_10_to_60_combinations[idx_s] = append(sweeps_10_to_60_combinations[idx_s], r_combinations[0])
			// fmt.Println(idx_s, s, r_combinations[0])
			sweeps_10_to_60_durations[idx_s] = append(sweeps_10_to_60_durations[idx_s], r_durations[0])
			sweeps_10_to_60_revenue[idx_s] = append(sweeps_10_to_60_revenue[idx_s], r_revenue[0])
		}(idx_s, s)
	}
	c <- sweeps_10_to_60_combinations
	d <- sweeps_10_to_60_durations
	e <- sweeps_10_to_60_revenue
}

func get_min_duration_across_geos() int {
	duration_list := []int{}
	rotates_list := []int{}
	final_duration := []int{}
	var min_duration int
	for _, geo := range geo_list {
		for i, v := range input_data[geo]["duration"] {
			v_int, _ := strconv.Atoi(v)
			r_int, _ := strconv.Atoi(input_data[geo]["rotates"][i])
			duration_list = append(duration_list, v_int)
			rotates_list = append(rotates_list, r_int)
		}

		for i, v := range rotates_list {
			if v != 0 {
				final_duration = append(final_duration, duration_list[i])
			}
		}
		_, min_duration = min_in_slice_int(final_duration)
	}
	return min_duration
}

func deep_copy_args_map(a map[string][][]string) map[string][][]string {
	b := map[string][][]string{}
	for k, v := range a {
		switch reflect.ValueOf(v).Kind() {
		case reflect.Slice:
			val := make([][]string, len(v))
			copy(val, v)
			b[k] = val
		default:
			b[k] = v
		}
	}
	return b
}

func create_tree(sweeps_10_to_60_combinations [][][]int, sweeps_10_to_60_durations [][]int, sweeps_10_to_60_revenue [][]int,
	sweeps_10_to_60_margin [][]int, combination_tuple_key [][][]int, combination_tuple_value []int, args map[string][][]string) {

	min_duration := get_min_duration_across_geos()
	sweeps_10_to_60_revenue_temp := make([]int, len(sweeps_10_to_60_revenue))
	sweeps_10_to_60_durations_temp := make([]int, len(sweeps_10_to_60_durations))
	sweeps_10_to_60_margin_temp := make([]int, len(sweeps_10_to_60_margin))
	min_duration_index := 0
	for i, _ := range sweeps_10_to_60_revenue {
		sweeps_10_to_60_revenue_temp[i] = sum_elements_in_slice(sweeps_10_to_60_revenue[i])
		sweeps_10_to_60_durations_temp[i] = sum_elements_in_slice(sweeps_10_to_60_durations[i])
		sweeps_10_to_60_margin_temp[i] = sweeps_10_to_60_revenue_temp[i] - ((sweep_duration_list[i] / 10.0) * average_cost_per_10_sec)
	}

	for i, duration := range sweep_duration_list {
		if duration < min_duration {
			min_duration_index = i
			break
		}
	}
	sum_of_combination := 0
	for idx, comb := range sweeps_10_to_60_combinations {
		s := []int{}
		for _, c := range comb {
			s = append(s, sum_elements_in_slice(c))
		}
		sum_of_combination = sum_elements_in_slice(s)
		flag := false
		for idx_c, caption_combination := range comb {
			for idx_r, rotates := range caption_combination {
				a, _ := strconv.Atoi(args["min_rotates_list"][idx_c][idx_r])
				if a > 0 && rotates > 0 {
					flag = true
					break
				}
			}
			if flag {
				break
			}
		}
		if flag == false {
			sweeps_10_to_60_margin_temp[idx] = -9999999
		}
		if sum_of_combination == 0 {
			sweeps_10_to_60_margin_temp[idx] = -9999999
		}
	}
	max_margin_index, _ := max_in_slice_int(sweeps_10_to_60_margin_temp[min_duration_index:])
	combination_index := max_margin_index

	sweeps_to_be_pruned := sweeps_10_to_60_combinations[combination_index:]
	duration_to_be_pruned := sweeps_10_to_60_durations_temp[combination_index:]
	margin_to_be_pruned := sweeps_10_to_60_margin_temp[combination_index:]
	before_pruning += len(sweeps_to_be_pruned)
	negative_unique_non_zero_combinations, _, negative_unique_non_zero_margin := prune_sweeps(sweeps_to_be_pruned, duration_to_be_pruned, margin_to_be_pruned, args)
	after_pruning += len(negative_unique_non_zero_combinations)
	sum_branch_num += len(negative_unique_non_zero_combinations)
	branch_func_counter += 1
	if len(negative_unique_non_zero_combinations) > maxbranch {
		maxbranch = len(negative_unique_non_zero_combinations)
	}

	if len(negative_unique_non_zero_combinations) > 0 {
		for idx_c, combination := range negative_unique_non_zero_combinations {
			args_deep := deep_copy_args_map(args)
			min_equals_max := true

			for geo_index, _ := range geo_list {
				for idx, rotates := range args_deep["rotates_list"][geo_index] {
					p, _ := strconv.Atoi(rotates)
					q, _ := strconv.Atoi(args_deep["min_rotates_list"][geo_index][idx])
					r, _ := strconv.Atoi(args_deep["max_rotates_list"][geo_index][idx])

					args_deep["rotates_list"][geo_index][idx] = strconv.Itoa(p - combination[geo_index][idx])
					args_deep["min_rotates_list"][geo_index][idx] = strconv.Itoa(q - combination[geo_index][idx])
					args_deep["max_rotates_list"][geo_index][idx] = strconv.Itoa(r - combination[geo_index][idx])
				}
			}

			for geo_index, _ := range geo_list {
				for i, _ := range args_deep["rotates_list"][geo_index] {
					m, _ := strconv.Atoi(args_deep["min_rotates_list"][geo_index][i])
					if m > 0 {
						min_equals_max = false
						break
					}
				}

				if !min_equals_max {
					break
				}
			}

			if !min_equals_max {
				combination_tuple_key = append(combination_tuple_key, combination)
				combination_tuple_value = append(combination_tuple_value, negative_unique_non_zero_margin[idx_c])
				schedule_planner(false, combination_tuple_key, combination_tuple_value, args_deep)
			} else {
				fmt.Println(combination_tuple_key)
			}
		}
	} else {
		fmt.Println("In Else Else")
	}
}

func prune_sweeps(sweeps_to_be_pruned [][][]int, duration_to_be_pruned []int, margin_to_be_pruned []int, args map[string][][]string) ([][][]int, []int, []int) {
	negative_unique_non_zero_combinations := [][][]int{}
	negative_unique_non_zero_duration := []int{}
	negative_unique_non_zero_margin := []int{}
	negative_unique_non_zero_combinations_temp := [][][]int{}
	negative_unique_non_zero_duration_temp := []int{}
	negative_unique_non_zero_margin_temp := []int{}
	negative_unique_non_zero_min_constraint_duration := []int{}
	sweeps_satisfying_min_temp := [][][]int{}
	margin_satisfying_min_temp := []int{}
	duration_satisfying_min_temp := []int{}

	for s_index, sweep := range sweeps_to_be_pruned {
		for geo_index, geo := range sweep {
			sweep_added_flag := false
			for r_index, rotate := range geo {
				a, _ := strconv.Atoi(args["min_rotates_list"][geo_index][r_index])
				if rotate >= 1 && a >= 1 {
					sweeps_satisfying_min_temp = append(sweeps_satisfying_min_temp, sweep)
					margin_satisfying_min_temp = append(margin_satisfying_min_temp, margin_to_be_pruned[s_index])
					duration_satisfying_min_temp = append(duration_satisfying_min_temp, duration_to_be_pruned[s_index])
					sweep_added_flag = true
					break
				}
			}
			if sweep_added_flag {
				break
			}
		}
	}
	sweeps_to_be_pruned = sweeps_satisfying_min_temp
	duration_to_be_pruned = duration_satisfying_min_temp
	margin_to_be_pruned = margin_satisfying_min_temp
	best_case_min_constraint_index, _ := max_in_slice_int(margin_to_be_pruned)

	best_case_min_constraint_duration := 0
	for geo_index, geo := range sweeps_to_be_pruned[best_case_min_constraint_index] {
		for r_index, rotates := range geo {
			a, _ := strconv.Atoi(args["min_rotates_list"][geo_index][r_index])
			b, _ := strconv.Atoi(args["duration_list"][geo_index][r_index])
			if a > 0 && rotates > 0 {
				best_case_min_constraint_duration += b * rotates
			}
		}
	}

	for x, _ := range sweeps_to_be_pruned {
		combination_min_constraint_duration := 0
		combination := sweeps_to_be_pruned[x]
		duration := duration_to_be_pruned[x]
		revenue := margin_to_be_pruned[x]
		s := []int{}
		for _, v := range combination {
			s = append(s, sum_elements_in_slice(v))
		}
		sum_of_combination := sum_elements_in_slice(s)
		if sum_of_combination > 0 {
			helps_meet_min_constraint := false
			for idx, caption_combination := range combination {
				for idx_r, rotates := range caption_combination {
					a, _ := strconv.Atoi(args["min_rotates_list"][idx][idx_r])
					b, _ := strconv.Atoi(args["duration_list"][idx][idx_r])
					if a > 0 && rotates > 0 {
						helps_meet_min_constraint = true
						combination_min_constraint_duration += b * rotates
					}
				}
			}

			if helps_meet_min_constraint && combination_min_constraint_duration >= best_case_min_constraint_duration {
				negative_unique_non_zero_combinations_temp = append(negative_unique_non_zero_combinations_temp, combination)
				negative_unique_non_zero_duration_temp = append(negative_unique_non_zero_duration_temp, duration)
				negative_unique_non_zero_margin_temp = append(negative_unique_non_zero_margin_temp, revenue)
				negative_unique_non_zero_min_constraint_duration = append(negative_unique_non_zero_min_constraint_duration, combination_min_constraint_duration)
			}

			for idx, duration := range negative_unique_non_zero_min_constraint_duration {
				if count_elements_in_slice(negative_unique_non_zero_min_constraint_duration, duration) > 1 {
					max_margin := negative_unique_non_zero_margin_temp[idx]
					max_index := idx
					for idx_m, duration_m := range negative_unique_non_zero_min_constraint_duration {
						current_max_margin := negative_unique_non_zero_margin_temp[idx_m]
						if duration == duration_m && current_max_margin > max_margin {
							max_margin = current_max_margin
							max_index = idx_m
						}
					}

					selected_combination := negative_unique_non_zero_combinations_temp[max_index]
					selected_duration := negative_unique_non_zero_duration_temp[max_index]
					selected_margin := negative_unique_non_zero_margin_temp[max_index]
					if sliceInSlice(selected_combination, negative_unique_non_zero_combinations) == false {
						negative_unique_non_zero_combinations = append(negative_unique_non_zero_combinations, selected_combination)
						negative_unique_non_zero_duration = append(negative_unique_non_zero_duration, selected_duration)
						negative_unique_non_zero_margin = append(negative_unique_non_zero_margin, selected_margin)
					}
				} else {
					selected_combination := negative_unique_non_zero_combinations_temp[idx]
					selected_duration := negative_unique_non_zero_duration_temp[idx]
					selected_margin := negative_unique_non_zero_margin_temp[idx]
					if sliceInSlice(selected_combination, negative_unique_non_zero_combinations) == false {
						negative_unique_non_zero_combinations = append(negative_unique_non_zero_combinations, selected_combination)
						negative_unique_non_zero_duration = append(negative_unique_non_zero_duration, selected_duration)
						negative_unique_non_zero_margin = append(negative_unique_non_zero_margin, selected_margin)
					}
				}
			}
		}
	}
	return negative_unique_non_zero_combinations, negative_unique_non_zero_duration, negative_unique_non_zero_margin
}

func main() {
	flag.Parse()

	// Reading input from file which would be provided by Rajesh"s API (node.js)
	dat, _ := ioutil.ReadFile("input.txt")

	err := json.Unmarshal(dat, &input_data)

	if err != nil {
		panic(err)
	}

	// Creating list of all the geos from the input data to maintain order
	//(cannot loop over input_data as it is a hash map)
	geo_list = make([]string, len(input_data))
	i := 0
	for k := range input_data {
		geo_list[i] = k
		i++
	}

	for i := *skip; i <= *max_spot_length; i += *skip {
		sweep_duration_list = append(sweep_duration_list, i)
	}

	args := map[string][][]string{}
	check := true
	combination_tuple_key := [][][]int{}
	combination_tuple_value := []int{}
	schedule_planner(check, combination_tuple_key, combination_tuple_value, args)
	wg.Wait()
}
