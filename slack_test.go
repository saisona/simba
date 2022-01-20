package simba_test

import (
	"testing"

	"github.com/saisona/simba"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestAddingContextAuthor(t *testing.T) {
	contextBlock := simba.AddingContextAuthor("fake_author")
	assert.Contains(t, contextBlock.BlockID, "author_context_qotd_", "BlockId wrong")
	assert.Len(t, contextBlock.ContextElements.Elements, 1, "Should always have only 1 elem")
}

func TestContextInputText(t *testing.T) {
	contextInput := simba.ContextInputText()
	assert.Equal(t, contextInput.BlockID, "MoodContext", "Should be MoodContext")
	assert.NotEqual(t, contextInput.Element, nil)
	assert.IsType(t, &slack.PlainTextInputBlockElement{}, contextInput.Element)
	assert.False(t, contextInput.DispatchAction)
	assert.True(t, contextInput.Optional)

	elem := contextInput.Element.(*slack.PlainTextInputBlockElement)
	assert.Equal(t, elem.ActionID, "mood_ctxt")
}

func TestFromFeelingToSmiley(t *testing.T) {
	assert.Equal(t, ":meow:", simba.FromFeelingToSmiley(""))
	assert.Equal(t, ":star-struck:", simba.FromFeelingToSmiley("Excited"))
	assert.Equal(t, ":smile:", simba.FromFeelingToSmiley("Happy"))
	assert.Equal(t, ":relaxed:", simba.FromFeelingToSmiley("Chilling"))
	assert.Equal(t, ":expressionless:", simba.FromFeelingToSmiley("Neutral"))
	assert.Equal(t, ":face_with_rolling_eyes:", simba.FromFeelingToSmiley("Frustrated"))
	assert.Equal(t, ":yawning_face:", simba.FromFeelingToSmiley("Tired"))
	assert.Equal(t, ":cry:", simba.FromFeelingToSmiley("Sad"))
	assert.Equal(t, ":triumph:", simba.FromFeelingToSmiley("Mad"))
	assert.Equal(t, ":disappointed:", simba.FromFeelingToSmiley("Disappointed"))
}

func TestFromMoodToSmiley(t *testing.T) {
	assert.Equal(t, ":meow:", simba.FromMoodToSmiley(""))
	assert.Equal(t, ":heart:", simba.FromMoodToSmiley("good_mood"))
	assert.Equal(t, ":yellow_heart:", simba.FromMoodToSmiley("average_mood"))
	assert.Equal(t, ":black_heart:", simba.FromMoodToSmiley("bad_mood"))
}

func TestDrawResultsEmpty(t *testing.T) {
	blocks, err := simba.DrawResults([]*simba.User{})
	if err != nil {
		t.Error(err)
	}
	assert.Len(t, blocks, 0)
}

func TestDrawResultsOneUserNoMood(t *testing.T) {
	fakeSimbaUser := &simba.User{
		Model:          gorm.Model{ID: 1},
		SlackUserID:    "fake_XXX",
		SlackChannelId: "fake_channel_XXX",
		IsManager:      false,
		Username:       "fake_username",
		Moods:          []simba.DailyMood{},
	}
	blocks, err := simba.DrawResults([]*simba.User{fakeSimbaUser})
	if err != nil {
		t.Error(err)
	}
	assert.Len(t, blocks, 0)
}

func TestDrawResultsOneUserOneMoodNoFeelingNoContext(t *testing.T) {
	fakeMood := simba.DailyMood{
		Model:  gorm.Model{ID: 1},
		UserID: 1,
		Mood:   "good_mood",
	}
	fakeSimbaUser := &simba.User{
		Model:          gorm.Model{ID: 1},
		SlackUserID:    "fake_XXX",
		SlackChannelId: "fake_channel_XXX",
		IsManager:      false,
		Username:       "fake_username",
		Moods:          []simba.DailyMood{fakeMood},
	}
	slackBlocks, err := simba.DrawResults([]*simba.User{fakeSimbaUser})
	if err != nil {
		t.Error(err)
	}
	assert.Len(t, slackBlocks, 1)
	anyBlock := slackBlocks[0]
	assert.IsType(t, &slack.SectionBlock{}, anyBlock)
	sectionBlock, ok := anyBlock.(*slack.SectionBlock)
	if !ok {
		t.FailNow()
	}
	assert.Nil(t, sectionBlock.Text)
	assert.Nil(t, sectionBlock.Accessory)
	assert.Len(t, sectionBlock.Fields, 2)
	assert.Equal(t, sectionBlock.Fields[0].Text, "fake_username")
	assert.Equal(t, sectionBlock.Fields[1].Text, ":heart: GOOD MOOD")
}

func TestDrawResultsOneUserOneMoodOneFeelingNoContext(t *testing.T) {
	fakeMood := simba.DailyMood{
		Model:   gorm.Model{ID: 1},
		UserID:  1,
		Mood:    "good_mood",
		Feeling: "Happy",
	}
	fakeSimbaUser := &simba.User{
		Model:          gorm.Model{ID: 1},
		SlackUserID:    "fake_XXX",
		SlackChannelId: "fake_channel_XXX",
		IsManager:      false,
		Username:       "fake_username",
		Moods:          []simba.DailyMood{fakeMood},
	}
	slackBlocks, err := simba.DrawResults([]*simba.User{fakeSimbaUser})
	if err != nil {
		t.Error(err)
	}
	assert.Len(t, slackBlocks, 1)
	anyBlock := slackBlocks[0]
	assert.IsType(t, &slack.SectionBlock{}, anyBlock)
	sectionBlock := anyBlock.(*slack.SectionBlock)
	assert.Nil(t, sectionBlock.Text)
	assert.Nil(t, sectionBlock.Accessory)
	assert.Len(t, sectionBlock.Fields, 2)
	assert.Equal(t, sectionBlock.Fields[0].Text, "fake_username")
	assert.Equal(t, sectionBlock.Fields[1].Text, ":heart: :smile: Happy")
}

func TestDrawResultsOneUserOneMoodOneFeelingWithContext(t *testing.T) {
	fakeMood := simba.DailyMood{
		Model:   gorm.Model{ID: 1},
		UserID:  1,
		Mood:    "good_mood",
		Feeling: "Happy",
		Context: "Small one",
	}
	fakeSimbaUser := &simba.User{
		Model:          gorm.Model{ID: 1},
		SlackUserID:    "fake_XXX",
		SlackChannelId: "fake_channel_XXX",
		IsManager:      false,
		Username:       "fake_username",
		Moods:          []simba.DailyMood{fakeMood},
	}
	slackBlocks, err := simba.DrawResults([]*simba.User{fakeSimbaUser})
	if err != nil {
		t.Error(err)
	}
	assert.Len(t, slackBlocks, 2)
	anyBlock := slackBlocks[0]
	assert.IsType(t, &slack.SectionBlock{}, anyBlock)
	sectionBlock := anyBlock.(*slack.SectionBlock)
	assert.Nil(t, sectionBlock.Text)
	assert.Nil(t, sectionBlock.Accessory)
	assert.Len(t, sectionBlock.Fields, 2)
	assert.Equal(t, sectionBlock.Fields[0].Text, "fake_username")
	assert.Equal(t, sectionBlock.Fields[1].Text, ":heart: :smile: Happy")
	assert.IsType(t, &slack.ContextBlock{}, slackBlocks[1])

	anyBlock = slackBlocks[1]
	contextBlock := anyBlock.(*slack.ContextBlock)
	assert.Equal(t, contextBlock.BlockID, "context_1")
	assert.Len(t, contextBlock.ContextElements.Elements, 1)
	assert.IsType(t, &slack.TextBlockObject{}, contextBlock.ContextElements.Elements[0])

	anyMixedBlock := contextBlock.ContextElements.Elements[0].(*slack.TextBlockObject)
	assert.Equal(t, anyMixedBlock.Text, "Small one")
}

func TestDrawResultsTwoUsersOneMoodOneFeelingWithContext(t *testing.T) {
	fakeMood1 := simba.DailyMood{
		Model:   gorm.Model{ID: 1},
		UserID:  1,
		Mood:    "good_mood",
		Feeling: "Happy",
		Context: "Small one",
	}
	fakeMood2 := simba.DailyMood{
		Model:   gorm.Model{ID: 2},
		UserID:  2,
		Mood:    "bad_mood",
		Feeling: "Sad",
		Context: "Wanna cry",
	}
	fakeSimbaUser1 := &simba.User{
		Model:          gorm.Model{ID: 1},
		SlackUserID:    "fake_XXX",
		SlackChannelId: "fake_channel_XXX",
		IsManager:      false,
		Username:       "fake_username",
		Moods:          []simba.DailyMood{fakeMood1},
	}
	fakeSimbaUser2 := &simba.User{
		Model:          gorm.Model{ID: 2},
		SlackUserID:    "fake_XXX",
		SlackChannelId: "fake_channel_XXX",
		IsManager:      false,
		Username:       "fake_username2",
		Moods:          []simba.DailyMood{fakeMood2},
	}
	slackBlocks, err := simba.DrawResults([]*simba.User{fakeSimbaUser1, fakeSimbaUser2})
	if err != nil {
		t.Error(err)
	}
	assert.Len(t, slackBlocks, 5)
	anyBlock := slackBlocks[0]
	assert.IsType(t, &slack.SectionBlock{}, anyBlock)
	sectionBlock := anyBlock.(*slack.SectionBlock)
	assert.Nil(t, sectionBlock.Text)
	assert.Nil(t, sectionBlock.Accessory)
	assert.Len(t, sectionBlock.Fields, 2)
	assert.Equal(t, sectionBlock.Fields[0].Text, "fake_username")
	assert.Equal(t, sectionBlock.Fields[1].Text, ":heart: :smile: Happy")
	assert.IsType(t, &slack.ContextBlock{}, slackBlocks[1])

	anyBlock = slackBlocks[1]
	contextBlock := anyBlock.(*slack.ContextBlock)
	assert.Equal(t, contextBlock.BlockID, "context_1")
	assert.Len(t, contextBlock.ContextElements.Elements, 1)
	assert.IsType(t, &slack.TextBlockObject{}, contextBlock.ContextElements.Elements[0])

	anyMixedBlock := contextBlock.ContextElements.Elements[0].(*slack.TextBlockObject)
	assert.Equal(t, anyMixedBlock.Text, "Small one")

	assert.IsType(t, &slack.SectionBlock{}, slackBlocks[2])

	anyBlock2 := slackBlocks[3]
	assert.IsType(t, &slack.SectionBlock{}, anyBlock2)
	sectionBlock2 := anyBlock2.(*slack.SectionBlock)
	assert.Nil(t, sectionBlock2.Text)
	assert.Nil(t, sectionBlock2.Accessory)
	assert.Len(t, sectionBlock2.Fields, 2)
	assert.Equal(t, sectionBlock2.Fields[0].Text, "fake_username2")
	assert.Equal(t, sectionBlock2.Fields[1].Text, ":black_heart: :cry: Sad")

	assert.IsType(t, &slack.ContextBlock{}, slackBlocks[4])
	anyBlock2 = slackBlocks[4]
	contextBlock2 := anyBlock2.(*slack.ContextBlock)
	assert.Equal(t, contextBlock2.BlockID, "context_2")
	assert.Len(t, contextBlock2.ContextElements.Elements, 1)
	assert.IsType(t, &slack.TextBlockObject{}, contextBlock2.ContextElements.Elements[0])

	anyMixedBlock2 := contextBlock2.ContextElements.Elements[0].(*slack.TextBlockObject)
	assert.Equal(t, anyMixedBlock2.Text, "Wanna cry")
}
