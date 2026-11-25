package i18n

// The keys of the catalogue of controls, the view qadg-gui opens with
// -catalogue (DEC-253).
//
// # Why a file of their own
//
// i18n.go was past the ceiling on the length of a file when they came, and a
// ceiling may be lowered and never raised (the keys of the take screen left it
// for take.go for the same reason). These keys are a register of their own as
// well:
// they name the controls of the window and their states, and nothing a person
// working with data ever reads. What stands ON the controls of the catalogue
// is not here. The long text and the extreme values are samples, kept in
// internal/gui/screens beside the sample recipes, because nobody translates a
// specimen (DEC-253, point 2).
const (
	// CataloguePromise is the line under the name of the window in the
	// sidebar, where the window proper says it is offline. It says which of
	// the two frames this is.
	CataloguePromise Key = "catalogue.promise"

	// The heading the pages stand under in the column, and the pages, one per
	// family of controls.
	CatalogueGroup         Key = "catalogue.group"
	CataloguePageButtons   Key = "catalogue.page.buttons"
	CataloguePageChoosers  Key = "catalogue.page.choosers"
	CataloguePageChecks    Key = "catalogue.page.checks"
	CataloguePageText      Key = "catalogue.page.text"
	CataloguePageLines     Key = "catalogue.page.lines"
	CataloguePageLists     Key = "catalogue.page.lists"
	CataloguePageValues    Key = "catalogue.page.values"
	CataloguePageAreas     Key = "catalogue.page.areas"
	CataloguePageQuestions Key = "catalogue.page.questions"

	// The controls, one per type of internal/gui/parts a screen builds.
	CatalogueButton     Key = "catalogue.control.button"
	CatalogueSelect     Key = "catalogue.control.select"
	CatalogueCheck      Key = "catalogue.control.check"
	CatalogueTapRow     Key = "catalogue.control.taprow"
	CatalogueValueList  Key = "catalogue.control.values"
	CatalogueField      Key = "catalogue.control.field"
	CatalogueEditor     Key = "catalogue.control.editor"
	CatalogueTable      Key = "catalogue.control.table"
	CatalogueScroll     Key = "catalogue.control.scroll"
	CatalogueDisclosure Key = "catalogue.control.disclosure"
	CatalogueQuestion   Key = "catalogue.control.question"

	// The states a control is shown in, one per column.
	CatalogueStateRest     Key = "catalogue.state.rest"
	CatalogueStatePointer  Key = "catalogue.state.pointer"
	CatalogueStatePressed  Key = "catalogue.state.pressed"
	CatalogueStateFocus    Key = "catalogue.state.focus"
	CatalogueStateDisabled Key = "catalogue.state.disabled"

	// The variants a control is shown in, one per row.
	CatalogueOutlined     Key = "catalogue.variant.outlined"
	CatalogueIcon         Key = "catalogue.variant.icon"
	CatalogueFilled       Key = "catalogue.variant.filled"
	CataloguePlace        Key = "catalogue.variant.place"
	CatalogueCurrent      Key = "catalogue.variant.current"
	CatalogueLongLabel    Key = "catalogue.variant.long.label"
	CatalogueNothing      Key = "catalogue.variant.nothing"
	CatalogueChosen       Key = "catalogue.variant.chosen"
	CatalogueLongValue    Key = "catalogue.variant.long.value"
	CatalogueOff          Key = "catalogue.variant.off"
	CatalogueOn           Key = "catalogue.variant.on"
	CatalogueUnbanded     Key = "catalogue.variant.unbanded"
	CatalogueBanded       Key = "catalogue.variant.banded"
	CatalogueEmpty        Key = "catalogue.variant.empty"
	CatalogueHinted       Key = "catalogue.variant.hinted"
	CatalogueNoted        Key = "catalogue.variant.noted"
	CatalogueRefused      Key = "catalogue.variant.refused"
	CatalogueExtreme      Key = "catalogue.variant.extreme"
	CatalogueLines        Key = "catalogue.variant.lines"
	CatalogueMarkedLine   Key = "catalogue.variant.marked.line"
	CatalogueRows         Key = "catalogue.variant.rows"
	CatalogueClosed       Key = "catalogue.variant.closed"
	CatalogueOpened       Key = "catalogue.variant.opened"
	CatalogueOverflowed   Key = "catalogue.variant.overflowed"
	CatalogueThreeAnswers Key = "catalogue.variant.three.answers"
)

// catalogueKeys is every key the catalogue of controls may ask for.
var catalogueKeys = []Key{
	CataloguePromise, CatalogueGroup,
	CataloguePageButtons, CataloguePageChoosers, CataloguePageChecks, CataloguePageText,
	CataloguePageLines, CataloguePageLists, CataloguePageValues, CataloguePageAreas, CataloguePageQuestions,
	CatalogueButton, CatalogueSelect, CatalogueCheck, CatalogueTapRow, CatalogueValueList, CatalogueField,
	CatalogueEditor, CatalogueTable, CatalogueScroll, CatalogueDisclosure, CatalogueQuestion,
	CatalogueStateRest, CatalogueStatePointer, CatalogueStatePressed, CatalogueStateFocus,
	CatalogueStateDisabled,
	CatalogueOutlined, CatalogueIcon, CatalogueFilled, CataloguePlace, CatalogueCurrent,
	CatalogueLongLabel, CatalogueNothing, CatalogueChosen, CatalogueLongValue, CatalogueOff,
	CatalogueOn, CatalogueUnbanded, CatalogueBanded, CatalogueEmpty, CatalogueHinted, CatalogueNoted, CatalogueRefused,
	CatalogueExtreme, CatalogueLines, CatalogueMarkedLine, CatalogueRows, CatalogueClosed, CatalogueOpened,
	CatalogueOverflowed, CatalogueThreeAnswers,
}

// Keys is every key the window may ask for, the screens and the catalogue of
// controls together, so a test can close the register from the side the code
// is on.
var Keys = append(append([]Key{}, windowKeys...), catalogueKeys...)
