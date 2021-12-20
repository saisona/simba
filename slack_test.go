package simba_test

import (
	"strings"
	"testing"

	"github.com/saisona/simba"
	"github.com/slack-go/slack"
)

func TestAddingContextAuthor(t *testing.T) {
	contextBlock := simba.AddingContextAuthor("fake_author")
	if !strings.Contains(contextBlock.BlockID, "author_context_qotd_") {
		t.FailNow()
	} else if len(contextBlock.ContextElements.Elements) != 1 {
		t.FailNow()
	}
}

func TestContextInputText(t *testing.T) {
	contextInput := simba.ContextInputText()
	if contextInput.BlockID != "MoodContext" {
		t.Errorf("got %s, wanted MoodContext", contextInput.BlockID)
	} else if contextInput.Element == nil {
		t.Errorf("Element should not be nil")
	} else if elem, ok := contextInput.Element.(*slack.PlainTextInputBlockElement); !ok {
		t.Errorf("Element is not a *slack.PlainTextInputBlockElem")
	} else if elem.ActionID != "mood_ctxt" {
		t.Errorf("ActionId should be mood_ctxt, got %s", contextInput.Element.(slack.PlainTextInputBlockElement).ActionID)
	} else if contextInput.DispatchAction != false {
		t.Errorf("contextInput.DispatchAction should be false")
	}

}
