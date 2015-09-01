package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
)

var max_spot_length = flag.Int("s", 0, "max spot length")
var skip = flag.Int("skip", 5, "skip")
var sweep_duration_list = []int{}
var best_combination_by_geo = [][]int{}
var best_duration_by_geo = []int{}
var best_revenue_by_geo = []int{}

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
			best_combination_by_geo = append(best_combination_by_geo, 0)
		}

		best_duration_by_geo = append(best_duration_by_geo, 0)
		best_revenue_by_geo = append(best_revenue_by_geo, 0)

		return best_combination_by_geo, best_duration_by_geo, best_revenue_by_geo
	} else {
		weight := map[string]int{"margin": 1, "min": 100000}
	}
}

func sliceInSlice(a []int, list [][]int) bool {
	for _, b := range list {
		/* To-Do: Check if looping through the slice and
		comparing each element is  more performant than DeepEqual*/
		if reflect.DeepEqual(a, b) {
			return true
		}
	}
	return false
}

func main() {
	flag.Parse()
	// Reading input from file which would be provided by Rajesh"s API (node.js)
	dat, _ := ioutil.ReadFile("input.txt")
	var input_data map[string]map[string][]string
	err := json.Unmarshal(dat, &input_data)

	if err != nil {
		panic(err)
	}

	// Creating list of all the geos from the input data to maintain order
	//(cannot loop over input_data as it is a hash map)
	geo_list := make([]string, len(input_data))
	i := 0
	for k := range input_data {
		geo_list[i] = k
		i++
	}

	for i := *skip; i <= *max_spot_length; i += *skip {
		sweep_duration_list = append(sweep_duration_list, i)
	}

	for _, geo := range geo_list {
		fmt.Println("Key:", geo, "Value:", input_data[geo]["caption_names"])

		duration := input_data[geo]["duration"]
		rotates := input_data[geo]["rotates"]
		captions := input_data[geo]["captions"]
		// min_rotates := input_data[geo]["min_rotates"]
		max_rotates := input_data[geo]["max_rotates"]
		back_to_back := input_data[geo]["back_to_back"]
		back_to_back_max_rotates := input_data[geo]["back_to_back_max_rotates"]
		multiple_caption_combination := input_data[geo]["multiple_caption_combination"]
		effective_rate := input_data[geo]["effective_rate"]
		back_to_back_min_duration := input_data[geo]["back_to_back_min_duration"]

		// Step 1: Find all the possible ways a cation can be played
		flattened_rotates := flatten_rotates(rotates, max_rotates, back_to_back, back_to_back_max_rotates, back_to_back_min_duration, duration)
		flattened_combinations_pruned := cartesian_product(duration, flattened_rotates...)
		final_combinations := apply_multiple_caption_combination_constraint(flattened_combinations_pruned, captions, multiple_caption_combination)

		fmt.Println(final_combinations)
		for _, s := range sweep_duration_list {
			best_combination_by_geo = best_combination_by_geo[:0]
			best_duration_by_geo = best_duration_by_geo[:0]
			best_revenue_by_geo = best_revenue_by_geo[:0]

			filtered_combinations, filtered_durations, filtered_revenue := get_combinations_less_than_duration(final_combinations, duration, effective_rate, s)
			fmt.Println(filtered_combinations, filtered_durations, filtered_revenue, s)
		}

	}
}
