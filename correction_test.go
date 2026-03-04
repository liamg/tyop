package main

import "testing"

func TestCorrect(t *testing.T) {
	c := newCorrector(EnGB)

	tests := []struct {
		input    string
		expected string
	}{
		// --- ; → ' only when the result is a known word/contraction ---
		{"don;t worry", "don't worry"},
		{"won;t happen", "won't happen"},
		{"can;t do it", "can't do it"},
		{"it;s fine", "it's fine"},
		{"i;m here", "i'm here"},
		{"they;re coming", "they're coming"},
		{"you;re right", "you're right"},
		{"isn;t it", "isn't it"},
		{"doesn;t matter", "doesn't matter"},
		{"let;s go", "let's go"},
		// should NOT replace ; blindly
		{"semi;colon", "semi; colon"},
		{"key;value", "key; value"},

		// --- contractions without apostrophe ---
		{"im fine", "i'm fine"},
		{"dont worry", "don't worry"},
		{"cant do it", "can't do it"},
		{"wont work", "won't work"},
		{"theyre here", "they're here"},
		{"youre welcome", "you're welcome"},
		{"isnt it", "isn't it"},
		{"arent you", "aren't you"},
		{"didnt know", "didn't know"},
		{"doesnt matter", "doesn't matter"},
		{"wouldnt do it", "wouldn't do it"},
		{"couldnt help", "couldn't help"},

		// --- standalone i → I only when text has other capitals ---
		{"i think so", "i think so"},
		{"I think so", "I think so"},
		{"i am here. Steve knows.", "I am here. Steve knows."},

		// --- transpositions / spell check ---
		{"helol", "hello"},
		{"teh cat", "the cat"},
		{"recieve it", "receive it"},
		{"tset", "test"},
		{"can yopu fix this", "can you fix this"},
		{"the $100/mnth subscription", "the $100/month subscription"},
		{"what ifi do something", "what if i do something"},
		{"wel taht sckus", "well that sucks"},

		// --- spell corrections ---
		{"oh no, i maed a typo!", "oh no, i made a typo!"},

		// --- valid words must NOT be changed ---
		{"world", "world"},
		{"cat", "cat"},
		{"not now", "not now"},
		{"the quick brown fox", "the quick brown fox"},
		{"it's fine", "it's fine"},
		{"don't worry", "don't worry"},

		// --- proper nouns (capitalised) must NOT be spell-checked ---
		{"I went to Paris", "I went to Paris"},
		{"Thanks Steve", "Thanks Steve"},

		// --- sentence capitalisation (only when text already has capitals) ---
		{"hello world", "hello world"},
		{"hello world. how are you?", "hello world. how are you?"},
		{"I went to Paris. how nice.", "I went to Paris. How nice."},
		{"wait! what happened? nothing. I know.", "Wait! What happened? Nothing. I know."},
	}

	for _, tt := range tests {
		got := c.Correct(tt.input)
		if got != tt.expected {
			t.Errorf("Correct(%q)\n  got  %q\n  want %q", tt.input, got, tt.expected)
		}
	}
}


