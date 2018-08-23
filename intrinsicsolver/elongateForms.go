package intrinsicsolver

import (
	"regexp"
	"strings"
)

/* Function elongateForms is investigating for short-form functions and changes them for their long equivalent. */
func elongateForms(line *string, lines *[]string, idx int, name string) {
	var currentFunctions int
	pLines := *lines
	totalFunctions := strings.Count(*line, "!")
	for (currentFunctions != totalFunctions+1) && !strings.Contains(*line, "#!") && strings.Contains(*line, "!") {
		short := shortForm(name)
		long := longForm(name)
		full := fullForm(long)
		split := strings.Split(*line, short)
		if idx+1 < len(pLines) {
			if strings.Contains(*line, name) && strings.Contains(pLines[idx+1], "- ") && (len(split) != 2) {
				// If so - we don't have to surround it with quotes.
				if strings.Contains(*line, short) && !strings.Contains(*line, "|") {
					*line = strings.Replace(*line, short, full, -1)
				} else if strings.Contains(*line, short) && strings.Contains(*line, "|") {
					*line = strings.Replace(*line, (short + " |"), full, -1)
				}
			} else if strings.Contains(*line, name) {

				line = addQuotes(short, split, line)

				newFunctionForm := "\"" + long + "\":"
				newFunctionForm = SplitLinesIfNestedFunction(split, line, newFunctionForm)

				if strings.Contains(*line, short) && !strings.Contains(*line, "|") {
					*line = strings.Replace(*line, short, newFunctionForm, -1)
				} else if strings.Contains(*line, short) && strings.Contains(*line, "|") {
					*line = strings.Replace(*line, (short + " |"), newFunctionForm, -1)
				} else if strings.Contains(*line, full) && !strings.Contains(*line, "|") {
					*line = strings.Replace(*line, full, newFunctionForm, -1)
				} else if strings.Contains(*line, full) && strings.Contains(*line, "|") {
					*line = strings.Replace(*line, (full + " |"), newFunctionForm, -1)
				}
			}
		}
		currentFunctions++
	}
}

func SplitLinesIfNestedFunction(split []string, line *string, newFunctionForm string) string {
	//if this function is nested in the same line
	if len(split) > 1 && strings.Contains(split[0], "Fn::") {
		indent := 2 // can be anything >
		leadingSpaces := indent + countLeadingSpaces(*line)
		i := 0
		spaces := ""
		for i < leadingSpaces {
			spaces += " "
			i++
		}
		newFunctionForm = "\n" + spaces + newFunctionForm
	}
	return newFunctionForm
}

func addQuotes(short string, split []string, line *string) *string {
	// Function !Sub in its short form can take only a string - It has to be marked as string with quotes
	if short == "!Sub" {
		whiteSpaceTrimmed := strings.TrimSpace(split[1])
		if !regexp.MustCompile(`".*"`).MatchString(whiteSpaceTrimmed) && !strings.Contains(*line, "|") {
			*line = strings.Replace(*line, whiteSpaceTrimmed, ("\"" + whiteSpaceTrimmed + "\""), -1)
		}
	}
	return line
}

func countLeadingSpaces(line string) int {
	i := 0
	for _, runeValue := range line {
		if runeValue == ' ' {
			i++
		} else {
			break
		}
	}
	return i
}
