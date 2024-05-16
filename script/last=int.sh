#!/bin/bash

input_string="pod-1-1-13"
last_integer=$(echo "$input_string" | grep -oE '[0-9]+' | tail -1)
echo $last_integer  # Output: 0