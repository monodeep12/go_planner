package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
)

func flatten_rotates(rotates []string, max_rotates []string, back_to_back []string, back_to_back_max_rotates []string, back_to_back_min_duration []string, duration []string) [][]int {
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

func main() {
	var geo_list []string
	// Reading input from file which would be provided by Rajesh"s API (node.js)
	dat, _ := ioutil.ReadFile("input.txt")

	var input_data map[string]map[string][]string
	err := json.Unmarshal(dat, &input_data)

	if err != nil {
		panic(err)
	}

	// Creating list of all the geos from the input data to maintain order
	//(cannot loop over input_data as it is a hash map)
	for k := range input_data {
		geo_list = append(geo_list, k)
	}

	for _, geo := range geo_list {
		fmt.Println("Key:", geo, "Value:", input_data[geo]["caption_names"])

		duration := input_data[geo]["duration"]
		rotates := input_data[geo]["rotates"]
		// captions := input_data[geo]["captions"]
		// min_rotates := input_data[geo]["min_rotates"]
		max_rotates := input_data[geo]["max_rotates"]
		back_to_back := input_data[geo]["back_to_back"]
		back_to_back_max_rotates := input_data[geo]["back_to_back_max_rotates"]
		// multiple_caption_combination := input_data[geo]["multiple_caption_combination"]
		// effective_rate := input_data[geo]["effective_rate"]
		back_to_back_min_duration := input_data[geo]["back_to_back_min_duration"]

		// Step 1: Find all the possible ways a cation can be played
		flattened_rotates := flatten_rotates(rotates, max_rotates, back_to_back, back_to_back_max_rotates, back_to_back_min_duration, duration)
		fmt.Println(flattened_rotates)
	}
}
