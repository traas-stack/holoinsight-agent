/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package collectconfig

type (
	// TransformConf represents transform conf
	TransformConf struct {
		Filters []*TransformFilterConf `json:"filters" yaml:"filters"`
	}
	// TransformFilterConf transform filter
	TransformFilterConf struct {
		SwitchCaseV1 *TransformFilterSwitchCaseV1 `json:"switchCaseV1" yaml:"switchCaseV1"`
		SubstringV1  *TransformFilterSubstringV1  `json:"substringV1" yaml:"substringV1"`
		AppendV1     *TransformFilterAppendV1     `json:"appendV1" yaml:"appendV1"`
		CompositeV1  *TransformFilterCompositeV1  `json:"compositeV1" yaml:"compositeV1"`
		MappingV1    *TransformFilterMappingV1    `json:"mappingV1" yaml:"mappingV1"`
		// avoid using 'const' as field name or json field name, because 'const' is a keyword in some languages
		ConstV1         *TransformFilterConstV1         `json:"constV1" yaml:"constV1"`
		RegexpReplaceV1 *TransformFilterRegexpReplaceV1 `json:"regexpReplaceV1" yaml:"regexpReplaceV1"`
		DiscardV1       *struct{}                       `json:"discardV1" yaml:"discardV1"`
	}
	// TransformFilterAppendV1 represents appending suffix to the current value
	TransformFilterAppendV1 struct {
		Value           string `json:"value" yaml:"value"`
		AppendIfMissing bool   `json:"appendIfMissing" yaml:"appendIfMissing"`
	}
	// TransformFilterSubstringV1 represents extracting substring from the current value
	TransformFilterSubstringV1 struct {
		// Begin represents beginning offset of substring
		Begin int `json:"begin" yaml:"begin"`
		// End represents end offset of substring, -1 means match to string length
		End int `json:"end" yaml:"end"`
		// EmptyIfError represents returning "" if meet any error (e.g. begin >= len(str))
		EmptyIfError bool `json:"emptyIfError" yaml:"emptyIfError"`
	}
	// TransformFilterSwitchCaseV1 represents `switch/case/default` control flow.
	TransformFilterSwitchCaseV1 struct {
		// Cases will be tested in order, when find first matched case, its action will be executed, and then current filter process terminates.
		Cases []*TransformFilterSwitchCaseV1Case `json:"cases" yaml:"cases"`
		// DefaultAction will be executed if no case matches.
		DefaultAction *TransformFilterConf `json:"defaultAction" yaml:"defaultAction"`
	}
	// TransformFilterSwitchCaseV1Case represents one case and its action
	TransformFilterSwitchCaseV1Case struct {
		Case   *Where               `json:"caseWhere" yaml:"caseWhere"`
		Action *TransformFilterConf `json:"action" yaml:"action"`
	}
	// TransformFilterCompositeV1 is a container of filters. It executes filters in order.
	TransformFilterCompositeV1 struct {
		// Filters sub filters
		Filters []*TransformFilterConf `json:"filters" yaml:"filters"`
		// BreaksWhenError indicates whether to break the execution process when an error is encountered.
		// If it is false, it will print a log after encountering an error and ignore errors, and then continue to execute.
		BreaksWhenError bool `json:"breaksWhenError" yaml:"breaksWhenError"`
	}
	// TransformFilterMappingV1 represents mapping value from one to another
	TransformFilterMappingV1 struct {
		Mappings map[string]string `json:"mappings" yaml:"mappings"`
		// DefaultValue is used when mappings doesn't match source.
		// If DefaultValue is empty "", then the source value is returned.
		DefaultValue string `json:"defaultValue" yaml:"defaultValue"`
	}
	TransformFilterConstV1 struct {
		Value string `json:"value" yaml:"value"`
	}
	TransformFilterRegexpReplaceV1 struct {
		Expression  string `json:"expression,omitempty" yaml:"expression"`
		Replacement string `json:"replacement,omitempty" yaml:"replacement"`
	}
)
