package cli

import (
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// tooLongSeed is one character past what the surfaces take.
func tooLongSeed() string { return strings.Repeat("a", seed.Longest+1) }

// 🔴 THE GUARD IS ABOUT ALL THREE DOORS AT ONCE, and that is the whole point
// of it.
//
// A seed reaches this tool three ways: the --seed flag of quick, the --seed
// flag of generate, and the seed key of a recipe. This project has already paid
// for a rule written at one door and not carried to the next - a limit on rows
// that one entry point held and the other did not. Asking each door separately
// in its own test would let a fourth door be added with nobody noticing, so
// they are asked together and the list is the claim.
//
// What this cannot see: a door added later and not added here. Nothing in the
// code says how many doors there are, so the sentence above is the register.
func TestEveryDoorASeedArrivesThroughRefusesOneTooLong(t *testing.T) {
	long := tooLongSeed()

	t.Run("quick --seed", func(t *testing.T) {
		got := run("quick", "ssn", "--seed", long)
		refusesTheSeed(t, got)
	})

	t.Run("generate --seed", func(t *testing.T) {
		dir := t.TempDir()
		path := recipeIn(t, dir, smallRecipe(dir))
		got := run("generate", "--seed", long, path)
		refusesTheSeed(t, got)
	})

	t.Run("seed inside a recipe", func(t *testing.T) {
		dir := t.TempDir()
		body := strings.Replace(smallRecipe(dir), `seed: "test-seed"`, `seed: "`+long+`"`, 1)
		if !strings.Contains(body, long) {
			t.Fatal("the sample recipe no longer carries a seed line to replace")
		}
		got := run("validate", recipeIn(t, dir, body))
		if got.code == 0 {
			t.Fatalf("validate accepted a seed of %d characters", len([]rune(long)))
		}
		// Reported by the reader rather than by the run, so that validate
		// names it among all the other problems at once.
		if !strings.Contains(got.data+got.log, "longer than this tool takes") {
			t.Errorf("validate refused without saying the seed was too long:\n%s%s", got.data, got.log)
		}
	})
}

// refusesTheSeed checks the four parts a refusal of this project carries.
func refusesTheSeed(t *testing.T, got outcome) {
	t.Helper()
	if got.code == 0 {
		t.Fatalf("the run took a seed of more than %d characters and ended with 0", seed.Longest)
	}
	said := got.data + got.log
	for _, part := range []string{
		problemSeedLong,
		"hashed for every value",
		strconv.Itoa(seed.Longest) + " characters or fewer",
		problemSeedLongDo,
	} {
		if !strings.Contains(said, part) {
			t.Errorf("the refusal left out %q:\n%s", part, said)
		}
	}
}

// The limit is a limit, not a suggestion about round numbers: a seed exactly as
// long as it allows still works, and every dataset ever produced was made with
// one far shorter.
func TestASeedExactlyAtTheLimitIsStillTaken(t *testing.T) {
	got := run("quick", "ssn", "--seed", strings.Repeat("a", seed.Longest))
	if got.code != 0 {
		t.Fatalf("a seed of exactly %d characters was refused with %d: %s%s",
			seed.Longest, got.code, got.data, got.log)
	}
	if strings.TrimSpace(got.data) == "" {
		t.Error("a seed at the limit produced no value")
	}
}

// 🔴 The arithmetic behind the number, held where it cannot rot.
//
// The limit is not a taste: past the buffer the message is assembled in, every
// derived value costs an allocation instead of none, and there are as many
// derivations as there are values. The measurement is written beside the
// constant. This keeps the constant on the safe side of the cliff it was chosen
// for, so that raising it later has to be a decision rather than an edit.
//
// The fixed part of the message is 44 bytes and the four names share what is
// left, so a seed at the limit has to leave room for them.
func TestTheSeedLimitStaysInsideTheBufferItWasChosenFor(t *testing.T) {
	const messageRoom = 256 // seed.messageRoom, which is not exported
	const fixedPart = 44    // four lengths and two numbers
	room := messageRoom - fixedPart - seed.Longest
	if room <= 0 {
		t.Fatalf("a seed of %d characters leaves %d bytes for the entity, field and purpose names,"+
			" so every value would allocate", seed.Longest, room)
	}
	if room < 32 {
		t.Errorf("a seed of %d characters leaves only %d bytes for three names,"+
			" which a long entity name would use up", seed.Longest, room)
	}
}
