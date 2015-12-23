package antlr4

type ATN struct {
	DecisionToState      []*DecisionState
	grammarType          int
	maxTokenType         int
	states               []IATNState
	ruleToStartState     []*RuleStartState
	ruleToStopState      []*RuleStopState
	modeNameToStartState map[string]*TokensStartState
	modeToStartState     []*TokensStartState
	ruleToTokenType      []int
	lexerActions         []ILexerAction
}

func NewATN(grammarType int, maxTokenType int) *ATN {

	atn := new(ATN)

	// Used for runtime deserialization of ATNs from strings///
	// The type of the ATN.
	atn.grammarType = grammarType
	// The maximum value for any symbol recognized by a transition in the ATN.
	atn.maxTokenType = maxTokenType
	atn.states = make([]IATNState, 0)
	// Each subrule/rule is a decision point and we must track them so we
	//  can go back later and build DFA predictors for them.  This includes
	//  all the rules, subrules, optional blocks, ()+, ()* etc...
	atn.DecisionToState = make([]*DecisionState, 0)
	// Maps from rule index to starting state number.
	atn.ruleToStartState = make([]*RuleStartState, 0)
	// Maps from rule index to stop state number.
	atn.ruleToStopState = nil
	atn.modeNameToStartState = make(map[string]*TokensStartState)
	// For lexer ATNs, atn.maps the rule index to the resulting token type.
	// For parser ATNs, atn.maps the rule index to the generated bypass token
	// type if the
	// {@link ATNDeserializationOptions//isGenerateRuleBypassTransitions}
	// deserialization option was specified otherwise, atn.is {@code nil}.
	atn.ruleToTokenType = nil
	// For lexer ATNs, atn.is an array of {@link LexerAction} objects which may
	// be referenced by action transitions in the ATN.
	atn.lexerActions = nil
	atn.modeToStartState = make([]*TokensStartState, 0)

	return atn

}

// Compute the set of valid tokens that can occur starting in state {@code s}.
//  If {@code ctx} is nil, the set of tokens will not include what can follow
//  the rule surrounding {@code s}. In other words, the set will be
//  restricted to tokens reachable staying within {@code s}'s rule.
func (this *ATN) nextTokensInContext(s IATNState, ctx IRuleContext) *IntervalSet {
	var anal = NewLL1Analyzer(this)
	return anal.LOOK(s, nil, ctx)
}

// Compute the set of valid tokens that can occur starting in {@code s} and
// staying in same rule. {@link Token//EPSILON} is in set if we reach end of
// rule.
func (this *ATN) nextTokensNoContext(s IATNState) *IntervalSet {
	if s.getNextTokenWithinRule() != nil {
		return s.getNextTokenWithinRule()
	}
	s.setNextTokenWithinRule(this.nextTokensInContext(s, nil))
	s.getNextTokenWithinRule().readOnly = true
	return s.getNextTokenWithinRule()
}

func (this *ATN) nextTokens(s IATNState, ctx IRuleContext) *IntervalSet {
	if ctx == nil {
		return this.nextTokensNoContext(s)
	} else {
		return this.nextTokensInContext(s, ctx)
	}
}

func (this *ATN) addState(state IATNState) {
	if state != nil {
		state.setATN(this)
		state.SetStateNumber(len(this.states))
	}
	this.states = append(this.states, state)
}

func (this *ATN) removeState(state IATNState) {
	this.states[state.GetStateNumber()] = nil // just free mem, don't shift states in list
}

func (this *ATN) defineDecisionState(s *DecisionState) int {
	this.DecisionToState = append(this.DecisionToState, s)
	s.decision = len(this.DecisionToState) - 1
	return s.decision
}

func (this *ATN) getDecisionState(decision int) *DecisionState {
	if len(this.DecisionToState) == 0 {
		return nil
	} else {
		return this.DecisionToState[decision]
	}
}

// Computes the set of input symbols which could follow ATN state number
// {@code stateNumber} in the specified full {@code context}. This method
// considers the complete parser context, but does not evaluate semantic
// predicates (i.e. all predicates encountered during the calculation are
// assumed true). If a path in the ATN exists from the starting state to the
// {@link RuleStopState} of the outermost context without Matching any
// symbols, {@link Token//EOF} is added to the returned set.
//
// <p>If {@code context} is {@code nil}, it is treated as
// {@link ParserRuleContext//EMPTY}.</p>
//
// @param stateNumber the ATN state number
// @param context the full parse context
// @return The set of potentially valid input symbols which could follow the
// specified state in the specified context.
// @panics IllegalArgumentException if the ATN does not contain a state with
// number {@code stateNumber}

//var Token = require('./../Token').Token

func (this *ATN) getExpectedTokens(stateNumber int, ctx IRuleContext) *IntervalSet {
	if stateNumber < 0 || stateNumber >= len(this.states) {
		panic("Invalid state number.")
	}
	var s = this.states[stateNumber]
	var following = this.nextTokens(s, nil)
	if !following.contains(TokenEpsilon) {
		return following
	}
	var expected = NewIntervalSet()
	expected.addSet(following)
	expected.removeOne(TokenEpsilon)
	for ctx != nil && ctx.getInvokingState() >= 0 && following.contains(TokenEpsilon) {
		var invokingState = this.states[ctx.getInvokingState()]
		var rt = invokingState.getTransitions()[0]
		following = this.nextTokens(rt.(*RuleTransition).followState, nil)
		expected.addSet(following)
		expected.removeOne(TokenEpsilon)
		ctx = ctx.GetParent().(IRuleContext)
	}
	if following.contains(TokenEpsilon) {
		expected.addOne(TokenEOF)
	}
	return expected
}

var ATNINVALID_ALT_NUMBER = 0
