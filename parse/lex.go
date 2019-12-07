package parse

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

type item struct {
	typ itemType
	pos Position
	val string
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		return i.val
	case len(i.val) > 10:
		return fmt.Sprintf("%.10q...", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

type itemType int

const (
	itemError         itemType = iota // error occurred; value is text of error
	itemBool                          // boolean constant
	itemEquals                        // is equal ('==') check equality
	itemNotEquals                     // is not equal ('!=') check not equality
	itemGreaters                      // is greater ('>') check greater
	itemLowers                        // is lower ('<') check lower
	itemGreaterEquals                 // is greater equal ('>=') check greater equal
	itemLowerEquals                   // is lower equal ('<=') check lower equal
	itemNot                           // not ('!')
	itemOr                            // or ('||')
	itemAnd                           // or ('&&')
	itemAssign                        // equals ('=') introducing an assignment
	itemPow
	itemDiv
	itemAdd
	itemMinus
	itemMod
	itemEOF
	itemIdentifier
	itemFunction
	itemLeftRuleDelim       // left rule delimiter
	itemRightRuleDelim      // right rule delimiter
	itemLeftConditionDelim  // left condition delimiter
	itemRightConditionDelim // right condition delimiter
	itemLeftActionDelim     // left action delimiter
	itemRightActionDelim    // right action delimiter
	itemLeftFunctionDelim   // left function delimiter
	itemRightFunctionDelim  // right function delimiter
	itemLeftParen           // '(' inside action
	itemNumber              // simple number
	itemRightParen          // ')' inside action
	itemString              // quoted string (includes quotes)
	itemSeprator
	itemVariable
)

const eof = -1

type stateFn func(*lexer) stateFn

type lexer struct {
	input           []string // the string being scanned
	index           int
	pos             int       // current position in the input
	start           int       // start position of this item
	width           int       // width of last rune read from input
	items           chan item // channel of scanned items
	parenDepth      int       // nesting depth of ( ) exprs
	stateStack      []stateFn
	parenDepthStack []int
}

func (l *lexer) position() Position {
	return Position{
		Index: l.index,
		Char:  l.pos,
	}
}

func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input[l.index]) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.index][l.pos:])
	l.width = w
	l.pos += l.width
	return r
}

func (l *lexer) nextNonSpace() rune {
	r := l.next()
	for isSpace(r) {
		r = l.next()
	}
	return r
}

func (l *lexer) pushParenDepth() {
	l.parenDepthStack = append(l.parenDepthStack, l.parenDepth)
}

func (l *lexer) popParenDepth() int {
	if len(l.parenDepthStack) == 0 {
		return -1
	}
	lastStack := l.parenDepthStack[len(l.parenDepthStack)-1]
	l.parenDepthStack = l.parenDepthStack[0 : len(l.parenDepthStack)-1]
	return lastStack
}

func (l *lexer) popStackOnParenDepth() int {
	lastStack := l.popParenDepth()
	l.parenDepth = lastStack
	return lastStack
}

func (l *lexer) resetParenDepth() {
	l.parenDepth = 0
}

func (l *lexer) pushAndResetParenDepth() {
	l.pushParenDepth()
	l.resetParenDepth()
}

func (l *lexer) pushState(s stateFn) {
	l.stateStack = append(l.stateStack, s)
}

func (l *lexer) popState() stateFn {
	if len(l.stateStack) == 0 {
		return nil
	}
	lastStack := l.stateStack[len(l.stateStack)-1]
	l.stateStack = l.stateStack[0 : len(l.stateStack)-1]
	return lastStack
}

func (l *lexer) peekState() stateFn {
	lastStack := l.popState()
	if lastStack == nil {
		return nil
	}
	l.pushState(lastStack)
	return lastStack
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) peekNonSpace() rune {
	pos := l.pos
	w := l.width
	r := l.nextNonSpace()
	l.pos = pos
	l.width = w
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.position(), l.input[l.index][l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) ignoreSpace() {
	l.nextNonSpace()
	l.backup()
	l.ignore()
}

func (l *lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, l.position(), fmt.Sprintf(format, args...)}
	return nil
}

func (l *lexer) nextItem() item {
	return <-l.items
}

func (l *lexer) drain() {
	for range l.items {
	}
}

func lex(input []string) *lexer {
	l := &lexer{
		input: input,
		items: make(chan item),
	}
	go l.run()
	return l
}

func (l *lexer) run() {
	for state := lexBlock; state != nil; {
		state = state(l)
	}
	close(l.items)
}

const (
	delim = " | "
)

func lexBlock(l *lexer) stateFn {
	if len(l.input) > l.index {
		l.pos = 0
		l.width = 0
		l.start = 0
		l.emit(itemLeftRuleDelim)
		l.width = 0
		if x := strings.Index(l.input[l.index][l.pos:], delim); x >= 0 {
			return lexLeftOfCondition
		}
		return lexLeftOfAction
	}
	l.index--
	l.emit(itemEOF)
	l.index++
	return nil
}

func (l *lexer) atDelim() bool {
	return strings.HasPrefix(l.input[l.index][l.pos:], delim)
}

func lexLeftOfCondition(l *lexer) stateFn {
	l.emit(itemLeftConditionDelim)
	return lexInsideCondition
}

func lexLeftOfAction(l *lexer) stateFn {
	if l.atDelim() {
		l.pos += len(delim)
	}
	l.emit(itemLeftActionDelim)
	return lexInsideAction
}

func lexRightOfCondition(l *lexer) stateFn {
	l.emit(itemRightConditionDelim)
	return lexLeftOfAction
}

func lexRightOfAction(l *lexer) stateFn {
	l.emit(itemRightActionDelim)
	l.emit(itemRightRuleDelim)
	l.index++
	return lexBlock
}

func lexInsideCondition(l *lexer) stateFn {
	l.pushState(lexInsideCondition)
	return lexInsideExpression
}

func lexInsideExpression(l *lexer) stateFn {
	parentFn := l.peekState()
	if isStateFnEqual(parentFn, lexInsideCondition) {
		atDelim := l.atDelim()
		if atDelim {
			if l.parenDepth == 0 {
				l.popState()
				return lexRightOfCondition
			}
			return l.errorf("unclosed left paren")
		}
		if l.peek() == ',' {
			l.next()
			return l.errorf("unrecognized character in expression: %#U", ',')
		}
	}
	switch r := l.next(); {
	case r == eof:
		if isStateFnEqual(parentFn, lexInsideAction) {
			l.popState()
			if l.parenDepth != 0 || len(l.parenDepthStack) != 0 || len(l.stateStack) != 0 {
				return l.errorf("unexpected end of rule")
			}
			return lexRightOfAction
		}
		return l.errorf("unclosed expression")
	case isSpace(r):
		l.backup()
		return lexSpace
	case r == '!':
		if isAlphaNumeric(l.peek()) || l.peek() == '(' {
			l.emit(itemNot)
		} else if l.next() == '=' {
			l.emit(itemNotEquals)
		} else {
			return l.errorf("unrecognized character after '!': %#U", r)
		}
	case r == '>':
		if l.next() == '=' {
			l.emit(itemGreaterEquals)
		} else {
			l.backup()
			l.emit(itemGreaters)
		}
	case r == '<':
		if l.next() == '=' {
			l.emit(itemLowerEquals)
		} else {
			l.backup()
			l.emit(itemLowers)
		}
	case r == '=':
		if l.next() == '=' {
			l.emit(itemEquals)
		} else {
			l.backup()
			l.emit(itemAssign)
		}
	case r == '|':
		if l.next() != '|' {
			return l.errorf("expected ||")
		}
		l.emit(itemOr)
	case r == '&':
		if l.next() != '&' {
			return l.errorf("expected &&")
		}
		l.emit(itemAnd)
	case r == '+':
		if isSpace(l.peek()) {
			l.emit(itemAdd)
		}
	case r == '-':
		if isSpace(l.peek()) {
			l.emit(itemMinus)
		}
	case r == '*':
		if isSpace(l.peek()) {
			l.emit(itemPow)
		}
	case r == '/':
		if isSpace(l.peek()) {
			l.emit(itemDiv)
		}
	case r == '%':
		if isSpace(l.peek()) {
			l.emit(itemMod)
		}
	case r == '"':
		return lexQuote
	case r == '+' || r == '-' || ('0' <= r && r <= '9'):
		l.backup()
		return lexNumber
	case isAlphaNumeric(r):
		l.backup()
		return lexIdentifier
	case r == '(':
		l.emit(itemLeftParen)
		l.parenDepth++
	case r == ')':
		l.parenDepth--
		if l.parenDepth == 0 {
			if isStateFnEqual(parentFn, lexFunction) {
				l.emit(itemRightFunctionDelim)
				l.popStackOnParenDepth()
				l.popState()
				return lexInsideExpression
			}
		}
		if l.parenDepth < 0 {
			return l.errorf("unexpected right paren %#U", r)
		}
		l.emit(itemRightParen)
	case r == ',':
		l.emit(itemSeprator)
		if isStateFnEqual(parentFn, lexFunction) {
			if l.parenDepth > 1 {
				return l.errorf("unexpected end of parameter")
			}
		}
		if isStateFnEqual(parentFn, lexInsideAction) {
			if l.parenDepth != 0 {
				return l.errorf("unexpected end of action")
			}
			l.ignoreSpace()
			return l.popState()
		}
	default:
		return l.errorf("unrecognized character in expression: %#U", r)
	}
	return lexInsideExpression
}

func lexInsideAction(l *lexer) stateFn {
	if l.parenDepth != 0 {
		l.errorf("unclosed paran")
	}
Loop:
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r):
			// absorb.
		default:
			l.backup()
			word := l.input[l.index][l.start:l.pos]
			pos := l.pos
			w := l.width
			switch {
			case len(word) == 0:
				l.errorf("unexpected start of action")
			case l.nextNonSpace() == '=':
				if l.peekNonSpace() != '=' {
					l.pos = pos
					l.width = w
					l.emit(itemVariable)
					l.pushState(lexInsideAction)
				} else {
					l.errorf("no assignment found")
				}
			default:
				l.errorf("no assignment found")
			}
			break Loop
		}
	}
	return lexInsideExpression
}

func lexSpace(l *lexer) stateFn {
	var r rune
	var numSpaces int
	lexFn := l.peekState()
	for {
		r = l.peek()
		if !isSpace(r) {
			break
		}
		l.next()
		numSpaces++
	}

	if isStateFnEqual(lexFn, lexInsideCondition) {
		if strings.HasPrefix(l.input[l.index][l.pos-1:], delim) {
			l.backup() // Before the space.
		}
	}
	l.ignore()
	return lexInsideExpression
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isStateFnEqual(a, b stateFn) bool {
	return reflect.ValueOf(a) == reflect.ValueOf(b)
}

func lexNumber(l *lexer) stateFn {
	if !l.scanNumber() {
		return l.errorf("bad number syntax: %q", l.input[l.index][l.start:l.pos])
	}
	l.emit(itemNumber)
	return lexInsideExpression
}

func (l *lexer) scanNumber() bool {
	// Optional leading sign.
	l.accept("+-")
	// Is it hex?
	digits := "0123456789_"
	if l.accept("0") {
		// Note: Leading 0 does not mean octal in floats.
		if l.accept("xX") {
			digits = "0123456789abcdefABCDEF_"
		} else if l.accept("oO") {
			digits = "01234567_"
		} else if l.accept("bB") {
			digits = "01_"
		}
	}
	l.acceptRun(digits)
	if l.accept(".") {
		l.acceptRun(digits)
	}
	if len(digits) == 10+1 && l.accept("eE") {
		l.accept("+-")
		l.acceptRun("0123456789_")
	}
	if len(digits) == 16+6+1 && l.accept("pP") {
		l.accept("+-")
		l.acceptRun("0123456789_")
	}
	// Next thing mustn't be alphanumeric.
	if isAlphaNumeric(l.peek()) {
		l.next()
		return false
	}
	return true
}

func lexQuote(l *lexer) stateFn {
Loop:
	for {
		switch l.next() {
		case '\\':
			if r := l.next(); r != eof && r != '\n' {
				break
			}
			fallthrough
		case eof, '\n':
			return l.errorf("unterminated quoted string")
		case '"':
			break Loop
		}
	}
	l.emit(itemString)
	return lexInsideExpression
}

func lexIdentifier(l *lexer) stateFn {
Loop:
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r):
			// absorb.
		default:
			l.backup()
			word := l.input[l.index][l.start:l.pos]
			switch {
			case l.peek() == '(':
				l.emit(itemFunction)
				return lexFunction
			case word == "true", word == "false":
				l.emit(itemBool)
			default:
				l.emit(itemIdentifier)
			}
			break Loop
		}
	}
	return lexInsideExpression
}

func lexFunction(l *lexer) stateFn {
	l.next()
	l.emit(itemLeftFunctionDelim)
	l.pushAndResetParenDepth()
	l.pushState(lexFunction)
	l.parenDepth++
	return lexInsideExpression
}
