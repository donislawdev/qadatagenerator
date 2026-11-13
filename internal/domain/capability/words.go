package capability

// What each capability is, said for a person reading why a damage did not
// happen.
//
// # Why words and not the name
//
// A refusal reaches the manifest, the log and Check results, and the person
// reading it has never seen the catalog. "this run cannot schema.surname" names
// something only the catalog defines, and the names are not stable: the
// published catalog says so. So a refusal says what was missing in words, and
// the name stays where it is defined.
//
// # Why these are not the descriptions in the catalog
//
// Those are written for somebody implementing the catalog in another tool, as
// instructions, and several of them are not sentences. These are written for
// somebody whose run was refused, and each one fits the refusal it goes into.
//
// # What every phrase has to keep true
//
// A run phrase reads after "this damage needs", and a column phrase after
// "a column", so each fits both the sentence about a run and the one about the
// column the damage landed on. Several missing at once are joined into one list
// with commas and a final and, so a phrase holds neither outside the brackets
// its example sits in. Measured on the binary: with the examples after a comma,
// three phrases ran into each other and read as five. An example names a scheme only when a provider
// of this build supplies the capability and carries a check digit, and a test
// holds every example against the providers. The check digit is what lets a
// file be read as holding that scheme at all: a social security number
// supplies two of these and is never read out of a file, so naming it would
// send somebody with a column of them looking for what is already there. A
// test also holds that every capability in the registry has its words, so a
// new capability cannot reach a refusal as a bare name.

// runWords says what each capability that belongs to a run is.
var runWords = map[Name]string{
	OutputCharsetControl:    "a way to write the file in a character encoding other than the one it declares",
	OutputLineEndingControl: "a way to choose the line ending of each line",
	OutputByteLevelWrite:    "a way to write bytes that are not valid text",
	OutputBOMControl:        "a way to add or remove a byte order mark",
	FormatCSV:               "output written as CSV",
	FormatJSON:              "output written as JSON",

	SchemaFieldType:      "a declared type for every column",
	SchemaFieldRelations: "columns bound by a rule (such as two dates in order)",
	RelationForeignKey:   "a table that points at another table",
	KeyPrimary:           "a table with a declared primary key",

	IdentifierCheckDigit:       "an identifier with a check digit (such as a PESEL)",
	IdentifierEncodesDate:      "an identifier that encodes a date (such as a PESEL)",
	IdentifierEncodesGender:    "an identifier that encodes a gender (such as a PESEL)",
	IdentifierReservedRange:    "an identifier with a block that is never issued (such as an EAN barcode number)",
	IdentifierUnassignedTarget: "an identifier with a block nobody is assigned (such as an ABA routing number)",
	ReferencePostalToPlace:     "a list of which postal codes belong to which places",
	ProviderEmail:              "e-mail addresses made by this tool",
	CurrencyMinorUnits:         "amounts of money in a currency this tool knows",
	DateArithmetic:             "dates worked out from a reference point (such as business days and leap days)",

	CorpusUnicode:            "a set of Unicode characters (such as emoji and invisible characters)",
	CorpusCoercibleValues:    "a set of values that spreadsheets turn into another type on sight",
	CorpusSentinelValues:     "a set of placeholder values (such as an all nines postal code)",
	CorpusNamesInternational: "a set of real names from many scripts",

	UnicodeNormalization: "a way to write text in another Unicode normalisation form",
}

// columnWords says what a column has to be for each capability that belongs to
// a column.
var columnWords = map[Name]string{
	SchemaMaxLength:      "with a declared maximum length",
	SchemaEnumValues:     "with a declared list of allowed values",
	SchemaRequiredFields: "declared as required",
	SchemaPersonName:     "known to hold given names or surnames",
	SchemaSurname:        "known to hold surnames",
	KeyUnique:            "declared as unique",
}

// Words says what a capability is, as the thing a damage needs: "a column known
// to hold surnames", "a way to write bytes that are not valid text".
//
// A name the registry does not know comes back as itself. That cannot reach a
// refusal from a catalog this build accepted, and showing the name is more
// honest than showing nothing.
func (n Name) Words() string {
	if words, found := columnWords[n]; found {
		return "a column " + words
	}
	if words, found := runWords[n]; found {
		return words
	}
	return string(n)
}

// ColumnWords says what a column has to be for a capability that belongs to a
// column: "known to hold surnames". It is empty for one that belongs to a run.
func (n Name) ColumnWords() string {
	return columnWords[n]
}
