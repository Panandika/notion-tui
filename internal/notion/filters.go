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

	filter notionapi.Filter
}

// Active reports whether the filter limits anything.
func (f RowFilter) Active() bool {
	return f.filter != nil
}

// Query builds the request for one page of rows, starting after cursor
// ("" for the first page). Notion applies the filter, so it also covers
// rows that have not been loaded yet.
func (f RowFilter) Query(cursor string) *notionapi.DatabaseQueryRequest {
	if f.filter == nil && cursor == "" {
		return nil
	}
	return &notionapi.DatabaseQueryRequest{Filter: f.filter, StartCursor: notionapi.Cursor(cursor)}
}

// FilterProperty is a property rows can be filtered by, with the
// conditions offered for it.
type FilterProperty struct {
	Name    string
	Type    notionapi.PropertyType
	Choices []RowFilter
}

// FilterProperties lists what the rows of a database can be filtered by:
// status, select, multi-select, checkbox and date properties, in that
// order and by name within a type. now decides what "today" means for
// dates.
func FilterProperties(database *notionapi.Database, now time.Time) []FilterProperty {
	if database == nil {
		return nil
	}
	var properties []FilterProperty
	for name, config := range database.Properties {
		property := FilterProperty{Name: name}
		add := func(choice string, filter notionapi.PropertyFilter) {
			filter.Property = name
			property.Choices = append(property.Choices, RowFilter{Label: name + " " + choice, Choice: choice, filter: filter})
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
			property.Choices = dateChoices(name, now)

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
// and end in the time zone of now.
func dateChoices(name string, now time.Time) []RowFilter {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	day := func(offset int) *notionapi.Date {
		date := notionapi.Date(today.AddDate(0, 0, offset))
		return &date
	}
	condition := func(c notionapi.DateFilterCondition) notionapi.PropertyFilter {
		return notionapi.PropertyFilter{Property: name, Date: &c}
	}
	between := func(from, before int) notionapi.Filter {
		return notionapi.AndCompoundFilter{
			condition(notionapi.DateFilterCondition{OnOrAfter: day(from)}),
			condition(notionapi.DateFilterCondition{Before: day(before)}),
		}
	}

	choices := []RowFilter{
		{Choice: "is before today", filter: condition(notionapi.DateFilterCondition{Before: day(0)})},
		{Choice: "is today", filter: between(0, 1)},
		{Choice: "is in the next 7 days", filter: between(0, 7)},
		{Choice: "is after today", filter: condition(notionapi.DateFilterCondition{OnOrAfter: day(1)})},
		{Choice: "is set", filter: condition(notionapi.DateFilterCondition{IsNotEmpty: true})},
		{Choice: "is empty", filter: condition(notionapi.DateFilterCondition{IsEmpty: true})},
	}
	for i := range choices {
		choices[i].Label = name + " " + choices[i].Choice
	}
	return choices
}
