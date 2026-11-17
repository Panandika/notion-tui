package notion

import (
	"sort"
	"time"

	"github.com/jomei/notionapi"
)

// RowFilter is one condition on one property that limits which rows of a
// database are listed. The zero value lists every row.
type RowFilter struct {
	// Label says what is shown, for example "Status is Done".
	Label string
	// Choice is the condition without the property name: "is Done".
	Choice string

	// build makes the condition Notion receives. It is given the time of
	// the request, so "today" is right even if the app ran past midnight.
	build func(now time.Time) notionapi.Filter
}

// Active reports whether the filter limits anything.
func (f RowFilter) Active() bool {
	return f.build != nil
}

// Query builds the request for one page of rows, starting after cursor
// ("" for the first page). Notion applies the filter, so it also covers
// rows that have not been loaded yet. now decides what "today" means for
// date conditions.
func (f RowFilter) Query(cursor string, now time.Time) *notionapi.DatabaseQueryRequest {
	if f.build == nil && cursor == "" {
		return nil
	}
	request := &notionapi.DatabaseQueryRequest{StartCursor: notionapi.Cursor(cursor)}
	if f.build != nil {
		request.Filter = f.build(now)
	}
	return request
}

// always wraps a condition that does not depend on the date.
func always(filter notionapi.Filter) func(time.Time) notionapi.Filter {
	return func(time.Time) notionapi.Filter { return filter }
}

// FilterProperty is a property rows can be filtered by, with the
// conditions offered for it.
type FilterProperty struct {
	Name    string
	Type    notionapi.PropertyType
	Choices []RowFilter
}

// FilterProperties lists what the rows of a database can be filtered by:
// status, select, date, checkbox and multi-select properties, in that
// order and by name within a type.
func FilterProperties(database *notionapi.Database) []FilterProperty {
	if database == nil {
		return nil
	}
	var properties []FilterProperty
	for name, config := range database.Properties {
		property := FilterProperty{Name: name}
		add := func(choice string, filter notionapi.PropertyFilter) {
			filter.Property = name
			property.Choices = append(property.Choices, RowFilter{Label: name + " " + choice, Choice: choice, build: always(filter)})
		}

		switch config := config.(type) {
		case *notionapi.StatusPropertyConfig:
			property.Type = notionapi.PropertyTypeStatus
			for _, option := range config.Status.Options {
				add("is "+option.Name, notionapi.PropertyFilter{Status: &notionapi.StatusFilterCondition{Equals: option.Name}})
			}
			add("is empty", notionapi.PropertyFilter{Status: &notionapi.StatusFilterCondition{IsEmpty: true}})

		case *notionapi.SelectPropertyConfig:
			property.Type = notionapi.PropertyTypeSelect
			for _, option := range config.Select.Options {
				add("is "+option.Name, notionapi.PropertyFilter{Select: &notionapi.SelectFilterCondition{Equals: option.Name}})
			}
			add("is empty", notionapi.PropertyFilter{Select: &notionapi.SelectFilterCondition{IsEmpty: true}})

		case *notionapi.MultiSelectPropertyConfig:
			property.Type = notionapi.PropertyTypeMultiSelect
			for _, option := range config.MultiSelect.Options {
				add("has "+option.Name, notionapi.PropertyFilter{MultiSelect: &notionapi.MultiSelectFilterCondition{Contains: option.Name}})
			}
			add("is empty", notionapi.PropertyFilter{MultiSelect: &notionapi.MultiSelectFilterCondition{IsEmpty: true}})

		case *notionapi.CheckboxPropertyConfig:
			property.Type = notionapi.PropertyTypeCheckbox
			add("is checked", notionapi.PropertyFilter{Checkbox: &notionapi.CheckboxFilterCondition{Equals: true}})
			// "equals: false" would be dropped from the request as an
			// empty value, so unchecked is asked for as "not true".
			add("is unchecked", notionapi.PropertyFilter{Checkbox: &notionapi.CheckboxFilterCondition{DoesNotEqual: true}})

		case *notionapi.DatePropertyConfig:
			property.Type = notionapi.PropertyTypeDate
			property.Choices = dateChoices(name)

		default:
			continue
		}
		properties = append(properties, property)
	}

	sort.SliceStable(properties, func(i, j int) bool {
		if a, b := propertyRank(properties[i].Type), propertyRank(properties[j].Type); a != b {
			return a < b
		}
		return properties[i].Name < properties[j].Name
	})
	return properties
}

// dateChoices offers the date conditions people ask for most. Days start
// and end at midnight in the time zone of the request.
func dateChoices(name string) []RowFilter {
	// day returns midnight offset days from now's date. Each boundary is
	// built from the calendar date, so a day without a midnight (a clock
	// change) shifts only itself.
	day := func(now time.Time, offset int) *notionapi.Date {
		date := notionapi.Date(time.Date(now.Year(), now.Month(), now.Day()+offset, 0, 0, 0, 0, now.Location()))
		return &date
	}
	condition := func(c notionapi.DateFilterCondition) notionapi.Filter {
		return notionapi.PropertyFilter{Property: name, Date: &c}
	}
	between := func(from, before int) func(time.Time) notionapi.Filter {
		return func(now time.Time) notionapi.Filter {
			return notionapi.AndCompoundFilter{
				condition(notionapi.DateFilterCondition{OnOrAfter: day(now, from)}),
				condition(notionapi.DateFilterCondition{Before: day(now, before)}),
			}
		}
	}

	choices := []RowFilter{
		{Choice: "is before today", build: func(now time.Time) notionapi.Filter {
			return condition(notionapi.DateFilterCondition{Before: day(now, 0)})
		}},
		{Choice: "is today", build: between(0, 1)},
		{Choice: "is in the next 7 days", build: between(0, 7)}, // Today and the six days after
		{Choice: "is after today", build: func(now time.Time) notionapi.Filter {
			return condition(notionapi.DateFilterCondition{OnOrAfter: day(now, 1)})
		}},
		{Choice: "is set", build: always(condition(notionapi.DateFilterCondition{IsNotEmpty: true}))},
		{Choice: "is empty", build: always(condition(notionapi.DateFilterCondition{IsEmpty: true}))},
	}
	for i := range choices {
		choices[i].Label = name + " " + choices[i].Choice
	}
	return choices
}
