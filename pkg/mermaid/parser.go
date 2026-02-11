package mermaid

import (
	"errors"
	"fmt"
	"strings"
)

// Parser performs recursive descent parsing of Mermaid syntax.
type Parser struct {
	tokens      []Token
	pos         int
	diagram     *Diagram
	subgraphSeq int
	stack       []string
	subgraphMap map[string]*Subgraph
}

// ParseError represents a parsing error with location.
type ParseError struct {
	Message string
	Line    int
	Column  int
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at line %d, column %d: %s", e.Line, e.Column, e.Message)
}

// Parse parses Mermaid syntax and returns a Diagram AST.
func Parse(input string) (*Diagram, error) {
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()

	// Filter out whitespace and comments for easier parsing
	filtered := make([]Token, 0, len(tokens))
	for _, t := range tokens {
		if t.Type != TokenWhitespace && t.Type != TokenComment {
			filtered = append(filtered, t)
		}
	}

	parser := &Parser{
		tokens:      filtered,
		pos:         0,
		diagram:     &Diagram{},
		subgraphMap: map[string]*Subgraph{},
	}

	if err := parser.parse(); err != nil {
		return nil, err
	}

	return parser.diagram, nil
}

func (p *Parser) parse() error {
	// Skip leading newlines
	p.skipNewlines()

	if p.isAtEnd() {
		return errors.New("empty diagram")
	}

	// Detect diagram type from first keyword
	token := p.current()
	if token.Type != TokenKeyword {
		return p.error("expected diagram type (flowchart, sequenceDiagram, etc.)")
	}

	switch strings.ToLower(token.Value) {
	case "graph", diagramFlowchartName:
		return p.parseFlowchart()
	case "sequencediagram":
		return p.parseSequenceDiagram()
	case "classdiagram":
		return p.parseClassDiagram()
	case "statediagram", "statediagram-v2":
		return p.parseStateDiagram()
	case "erdiagram":
		return p.parseERDiagram()
	case "pie":
		return p.parsePieDiagram()
	default:
		return p.error(fmt.Sprintf("unsupported diagram type: %s", token.Value))
	}
}

// ==== Flowchart Parsing ====

func (p *Parser) parseFlowchart() error {
	p.diagram.Type = DiagramFlowchart
	p.advance() // consume graph/flowchart

	// Optional direction
	if p.check(TokenDirection) {
		p.diagram.Direction = p.parseDirection()
		p.advance()
	}

	p.skipNewlines()

	// Parse flowchart body
	for !p.isAtEnd() {
		if err := p.parseFlowchartStatement(); err != nil {
			return err
		}
		p.skipNewlines()
	}

	return nil
}

func (p *Parser) parseDirection() Direction {
	switch strings.ToUpper(p.current().Value) {
	case "TB", "TD":
		return DirectionTB
	case "BT":
		return DirectionBT
	case "LR":
		return DirectionLR
	case "RL":
		return DirectionRL
	default:
		return DirectionTB
	}
}

func (p *Parser) parseFlowchartStatement() error {
	// Handle subgraph
	if p.check(TokenSubgraph) {
		return p.parseSubgraph()
	}

	// Handle end of subgraph
	if p.check(TokenEnd) {
		p.advance()
		return nil
	}

	// Node or edge definition
	if p.isFlowchartIdentifier() {
		return p.parseNodeOrEdge()
	}

	// Skip unknown tokens
	if !p.isAtEnd() {
		p.advance()
	}

	return nil
}

func (p *Parser) parseSubgraph() error {
	p.advance() // consume 'subgraph'

	subgraph := &Subgraph{}

	// Parse subgraph ID (can also be keywords like "loop", "note", etc.)
	if p.isFlowchartIdentifier() {
		subgraph.ID = p.current().Value
		subgraph.Label = subgraph.ID // Default label is the ID
		p.advance()
	}
	if subgraph.ID == "" {
		p.subgraphSeq++
		subgraph.ID = fmt.Sprintf("__subgraph_%d", p.subgraphSeq)
		subgraph.Label = subgraph.ID
	}

	// Optional label in brackets overrides the ID-based label
	if p.check(TokenLabel) {
		subgraph.Label = extractLabel(p.current().Value)
		p.advance()
	}
	if parentID := p.currentSubgraph(); parentID != "" {
		subgraph.ParentID = parentID
		p.addToSubgraph(parentID, subgraph.ID)
	}

	p.skipNewlines()

	// Parse direction if present
	if p.check(TokenKeyword) && p.current().Value == "direction" {
		p.advance()
		if p.check(TokenDirection) {
			subgraph.Direction = p.parseDirection()
			p.advance()
		}
	}

	p.diagram.Subgraphs = append(p.diagram.Subgraphs, subgraph)
	p.subgraphMap[subgraph.ID] = subgraph
	p.stack = append(p.stack, subgraph.ID)
	defer func() {
		p.stack = p.stack[:len(p.stack)-1]
	}()

	p.skipNewlines()

	// Parse subgraph contents until 'end'
	for !p.isAtEnd() && !p.check(TokenEnd) {
		if err := p.parseFlowchartStatement(); err != nil {
			return err
		}
		p.skipNewlines()
	}

	if p.check(TokenEnd) {
		p.advance()
	}

	return nil
}

func (p *Parser) parseNodeOrEdge() error {
	// Parse first node
	nodeID := p.current().Value
	p.advance()

	// Check for node label
	var nodeLabel string
	var nodeShape NodeShape
	if p.check(TokenLabel) {
		nodeLabel, nodeShape = p.parseNodeLabel()
		p.advance()
	}

	// Ensure node exists unless this is a direct reference to a subgraph.
	if nodeLabel != "" || !p.isSubgraphID(nodeID) {
		node := p.findOrCreateNode(nodeID)
		if nodeLabel != "" {
			node.Label = nodeLabel
			node.Shape = nodeShape
		}
	}

	// Check for edge
	if p.check(TokenArrow) {
		return p.parseEdgeChain(nodeID)
	}

	return nil
}

func (p *Parser) parseEdgeChain(fromID string) error {
	for p.check(TokenArrow) {
		arrow := p.current().Value
		p.advance()

		edge := &Edge{
			From: fromID,
		}

		// Parse edge type and arrows from the arrow string
		edge.Type, edge.ArrowStart, edge.ArrowEnd = parseArrowStyle(arrow)

		// Check for edge label: |text| or |multi word text|
		if p.check(TokenPipe) {
			p.advance()
			edge.Label = p.consumeUntilPipe()
			if p.check(TokenPipe) {
				p.advance()
			}
		}

		// Parse target node
		if !p.isFlowchartIdentifier() {
			return p.error("expected node identifier after arrow")
		}

		toID := p.current().Value
		p.advance()

		// Check for target node label
		if p.check(TokenLabel) {
			label, shape := p.parseNodeLabel()
			p.advance()
			toNode := p.findOrCreateNode(toID)
			toNode.Label = label
			toNode.Shape = shape
		} else if !p.isSubgraphID(toID) {
			p.findOrCreateNode(toID)
		}

		edge.To = toID
		p.diagram.Edges = append(p.diagram.Edges, edge)

		// Continue chain if there's another arrow
		fromID = toID
	}

	return nil
}

func (p *Parser) parseNodeLabel() (string, NodeShape) {
	value := p.current().Value
	return extractLabel(value), detectShape(value)
}

func (p *Parser) findOrCreateNode(id string) *Node {
	for _, node := range p.diagram.Nodes {
		if node.ID == id {
			p.addToSubgraph(p.currentSubgraph(), id)
			return node
		}
	}
	node := &Node{
		ID:    id,
		Label: id, // Default label is the ID
		Shape: ShapeRectangle,
	}
	p.diagram.Nodes = append(p.diagram.Nodes, node)
	p.addToSubgraph(p.currentSubgraph(), id)
	return node
}

func (p *Parser) currentSubgraph() string {
	if len(p.stack) == 0 {
		return ""
	}
	return p.stack[len(p.stack)-1]
}

func (p *Parser) addToSubgraph(subgraphID, memberID string) {
	if subgraphID == "" || memberID == "" {
		return
	}
	sg := p.subgraphMap[subgraphID]
	if sg == nil {
		return
	}
	for _, existing := range sg.Nodes {
		if existing == memberID {
			return
		}
	}
	sg.Nodes = append(sg.Nodes, memberID)
}

func (p *Parser) isSubgraphID(id string) bool {
	_, ok := p.subgraphMap[id]
	return ok
}

// ==== Sequence Diagram Parsing ====

func (p *Parser) parseSequenceDiagram() error {
	p.diagram.Type = DiagramSequence
	p.advance() // consume 'sequenceDiagram'
	p.skipNewlines()

	for !p.isAtEnd() {
		if err := p.parseSequenceStatement(); err != nil {
			return err
		}
		p.skipNewlines()
	}

	return nil
}

//nolint:gocyclo // explicit branching keeps parser straightforward.
func (p *Parser) parseSequenceStatement() error {
	switch {
	case p.check(TokenParticipant):
		return p.parseParticipant(ParticipantDefault)
	case p.check(TokenActor):
		return p.parseParticipant(ParticipantActor)
	case p.check(TokenNote):
		return p.parseNote()
	case p.check(TokenLoop):
		return p.parseLoop()
	case p.check(TokenAlt):
		return p.parseAlt()
	case p.check(TokenOpt):
		return p.parseOpt()
	case p.check(TokenPar):
		return p.parsePar()
	case p.check(TokenRect):
		return p.parseRect()
	case p.check(TokenEnd):
		p.advance()
		return nil
	case p.check(TokenKeyword) && p.current().Value == "activate":
		return p.parseActivate()
	case p.check(TokenKeyword) && p.current().Value == "deactivate":
		return p.parseDeactivate()
	case p.check(TokenIdentifier):
		return p.parseMessage()
	default:
		if !p.isAtEnd() {
			p.advance()
		}
		return nil
	}
}

// parseActivate parses an explicit activate statement.
func (p *Parser) parseActivate() error {
	p.advance() // consume 'activate'

	if !p.check(TokenIdentifier) {
		return p.error("expected participant name after activate")
	}

	participant := p.current().Value
	p.advance()
	p.findOrCreateParticipant(participant)

	// Create an activation record
	// StartMsgIndex is the current message count (activation starts after the previous message)
	activation := &Activation{
		Participant:   participant,
		StartY:        -1, // Will be set during layout
		EndY:          -1,
		StartMsgIndex: len(p.diagram.Messages), // Index of message before which activation starts
		EndMsgIndex:   -1,
	}
	p.diagram.Activations = append(p.diagram.Activations, activation)

	return nil
}

// parseDeactivate parses an explicit deactivate statement.
func (p *Parser) parseDeactivate() error {
	p.advance() // consume 'deactivate'

	if !p.check(TokenIdentifier) {
		return p.error("expected participant name after deactivate")
	}

	participant := p.current().Value
	p.advance()

	// Find the most recent activation for this participant and mark its end
	for i := len(p.diagram.Activations) - 1; i >= 0; i-- {
		act := p.diagram.Activations[i]
		if act.Participant == participant && act.EndMsgIndex == -1 {
			// EndMsgIndex is the current message count (activation ends after this message)
			p.diagram.Activations[i].EndMsgIndex = len(p.diagram.Messages)
			p.diagram.Activations[i].EndY = -2 // Sentinel for "deactivation parsed"
			break
		}
	}

	return nil
}

func (p *Parser) parseParticipant(ptype ParticipantType) error {
	p.advance() // consume 'participant' or 'actor'

	if !p.check(TokenIdentifier) {
		return p.error("expected participant name")
	}

	participant := &Participant{
		ID:   p.current().Value,
		Type: ptype,
	}
	p.advance()

	// Check for alias: participant A as Alice
	if p.check(TokenKeyword) && p.current().Value == "as" {
		p.advance()
		if p.check(TokenIdentifier) || p.check(TokenString) {
			participant.Alias = p.current().Value
			p.advance()
		}
	}

	p.diagram.Participants = append(p.diagram.Participants, participant)
	return nil
}

func (p *Parser) parseMessage() error {
	from := p.current().Value
	p.advance()

	// Ensure participant exists
	p.findOrCreateParticipant(from)

	if !p.check(TokenArrow) {
		return nil // Just a participant reference
	}

	arrow := p.current().Value
	p.advance()

	if !p.check(TokenIdentifier) {
		return p.error("expected participant after arrow")
	}

	to := p.current().Value
	p.advance()
	p.findOrCreateParticipant(to)

	msg := &Message{
		From: from,
		To:   to,
		Type: parseMessageType(arrow),
	}

	// Check for colon and label
	if p.check(TokenColon) {
		p.advance()
		msg.Label = p.consumeRestOfLine()
	}

	// Check for activation/deactivation markers
	if strings.HasSuffix(arrow, "+") {
		msg.Activate = true
	} else if strings.HasSuffix(arrow, "-") {
		msg.Deactivate = true
	}

	p.diagram.Messages = append(p.diagram.Messages, msg)
	return nil
}

func (p *Parser) parseNote() error {
	p.advance() // consume 'note'

	note := &Note{}

	// Parse position: right of, left of, over
	if p.check(TokenKeyword) {
		switch p.current().Value {
		case "right":
			note.Position = NoteRightOf
			p.advance()
			if p.check(TokenKeyword) && p.current().Value == "of" {
				p.advance()
			}
		case "left":
			note.Position = NoteLeftOf
			p.advance()
			if p.check(TokenKeyword) && p.current().Value == "of" {
				p.advance()
			}
		case "over":
			note.Position = NoteOver
			p.advance()
		}
	}

	// Parse participant(s)
	if p.check(TokenIdentifier) {
		note.Participant = p.current().Value
		p.advance()

		// Check for second participant (note over A,B)
		if p.check(TokenComma) {
			p.advance()
			if p.check(TokenIdentifier) {
				note.Participant2 = p.current().Value
				p.advance()
			}
		}
	}

	// Parse note text
	if p.check(TokenColon) {
		p.advance()
		note.Text = p.consumeRestOfLine()
	}

	p.diagram.Notes = append(p.diagram.Notes, note)
	return nil
}

func (p *Parser) parseLoop() error {
	p.advance() // consume 'loop'

	loop := &LoopBlock{}

	// Parse loop label
	loop.Label = p.consumeRestOfLine()
	p.skipNewlines()

	// Parse loop body until 'end'
	for !p.isAtEnd() && !p.check(TokenEnd) {
		if p.check(TokenIdentifier) {
			// Store messages before parsing
			msgsBefore := len(p.diagram.Messages)
			if err := p.parseMessage(); err != nil {
				return err
			}
			// Add new messages to loop
			for i := msgsBefore; i < len(p.diagram.Messages); i++ {
				loop.Messages = append(loop.Messages, p.diagram.Messages[i])
			}
		} else {
			if err := p.parseSequenceStatement(); err != nil {
				return err
			}
		}
		p.skipNewlines()
	}

	if p.check(TokenEnd) {
		p.advance()
	}

	p.diagram.Loops = append(p.diagram.Loops, loop)
	return nil
}

//nolint:gocognit,gocyclo // complex parsing logic mirrors mermaid grammar.
func (p *Parser) parseAlt() error {
	p.advance() // consume 'alt'

	alt := &AltBlock{
		Condition: p.consumeRestOfLine(),
	}
	p.skipNewlines()

	// Parse alt body
	for !p.isAtEnd() && !p.check(TokenEnd) && !p.check(TokenElse) {
		if p.check(TokenIdentifier) {
			msgsBefore := len(p.diagram.Messages)
			if err := p.parseMessage(); err != nil {
				return err
			}
			for i := msgsBefore; i < len(p.diagram.Messages); i++ {
				alt.Messages = append(alt.Messages, p.diagram.Messages[i])
			}
		}
		p.skipNewlines()
	}

	// Handle else clauses (mermaid supports multiple else blocks)
	var lastElse *AltBlock
	for p.check(TokenElse) {
		p.advance()
		elseBlock := &AltBlock{
			Condition: p.consumeRestOfLine(),
		}
		p.skipNewlines()

		for !p.isAtEnd() && !p.check(TokenEnd) && !p.check(TokenElse) {
			if p.check(TokenIdentifier) {
				msgsBefore := len(p.diagram.Messages)
				if err := p.parseMessage(); err != nil {
					return err
				}
				for i := msgsBefore; i < len(p.diagram.Messages); i++ {
					elseBlock.Messages = append(elseBlock.Messages, p.diagram.Messages[i])
				}
			}
			p.skipNewlines()
		}

		// Chain else blocks
		if lastElse != nil {
			lastElse.Else = elseBlock
		} else {
			alt.Else = elseBlock
		}
		lastElse = elseBlock
	}

	if p.check(TokenEnd) {
		p.advance()
	}

	p.diagram.AltBlocks = append(p.diagram.AltBlocks, alt)
	return nil
}

func (p *Parser) parseOpt() error {
	p.advance() // consume 'opt'
	// Similar to alt but without else
	p.consumeRestOfLine()
	p.skipNewlines()

	for !p.isAtEnd() && !p.check(TokenEnd) {
		if err := p.parseSequenceStatement(); err != nil {
			return err
		}
		p.skipNewlines()
	}

	if p.check(TokenEnd) {
		p.advance()
	}
	return nil
}

func (p *Parser) parsePar() error {
	p.advance() // consume 'par'
	p.consumeRestOfLine()
	p.skipNewlines()

	for !p.isAtEnd() && !p.check(TokenEnd) {
		if err := p.parseSequenceStatement(); err != nil {
			return err
		}
		p.skipNewlines()
	}

	if p.check(TokenEnd) {
		p.advance()
	}
	return nil
}

func (p *Parser) parseRect() error {
	p.advance() // consume 'rect'
	p.consumeRestOfLine()
	p.skipNewlines()

	for !p.isAtEnd() && !p.check(TokenEnd) {
		if err := p.parseSequenceStatement(); err != nil {
			return err
		}
		p.skipNewlines()
	}

	if p.check(TokenEnd) {
		p.advance()
	}
	return nil
}

func (p *Parser) findOrCreateParticipant(id string) *Participant {
	for _, part := range p.diagram.Participants {
		if part.ID == id || part.Alias == id {
			return part
		}
	}
	part := &Participant{ID: id, Type: ParticipantDefault}
	p.diagram.Participants = append(p.diagram.Participants, part)
	return part
}

// ==== Class Diagram Parsing ====

func (p *Parser) parseClassDiagram() error {
	p.diagram.Type = DiagramClass
	p.advance() // consume 'classDiagram'
	p.skipNewlines()

	for !p.isAtEnd() {
		if err := p.parseClassStatement(); err != nil {
			return err
		}
		p.skipNewlines()
	}

	return nil
}

func (p *Parser) parseClassStatement() error {
	if p.check(TokenClass) {
		return p.parseClassDefinition()
	}

	if p.check(TokenIdentifier) {
		// Could be a class member or relationship
		className := p.current().Value
		p.advance()

		// Check for 'o' which might start an aggregation relationship (o--)
		if p.check(TokenIdentifier) && p.current().Value == "o" {
			p.advance() // consume 'o'
			if p.check(TokenArrow) || p.check(TokenMinus) {
				return p.parseClassRelationshipWithPrefix(className, "o")
			}
			// Otherwise it's not a relationship, treat 'o' as a standalone class
			p.findOrCreateClass("o")
			return nil
		}

		// Check for relationship arrow
		if p.check(TokenArrow) || p.check(TokenLess) || p.check(TokenStar) {
			return p.parseClassRelationship(className)
		}

		// Check for member definition: ClassName : +method()
		if p.check(TokenColon) {
			p.advance()
			return p.parseClassMember(className)
		}
	}

	if !p.isAtEnd() {
		p.advance()
	}
	return nil
}

func (p *Parser) parseClassDefinition() error {
	p.advance() // consume 'class'

	if !p.check(TokenIdentifier) {
		return p.error("expected class name")
	}

	class := &Class{
		Name: p.current().Value,
	}
	p.advance()

	// Check for body: class Foo { ... }
	if p.check(TokenOpenBrace) {
		p.advance()
		p.skipNewlines()

		for !p.isAtEnd() && !p.check(TokenCloseBrace) {
			member := p.parseClassMemberLine()
			if member != nil {
				if strings.Contains(member.Name, "(") {
					class.Methods = append(class.Methods, *member)
				} else {
					class.Attributes = append(class.Attributes, *member)
				}
			} else {
				// Skip unknown tokens to avoid infinite loop
				if !p.isAtEnd() && !p.check(TokenCloseBrace) && !p.check(TokenNewline) {
					p.advance()
				}
			}
			p.skipNewlines()
		}

		if p.check(TokenCloseBrace) {
			p.advance()
		}
	}

	p.diagram.Classes = append(p.diagram.Classes, class)
	return nil
}

//nolint:gocognit,gocyclo // complex parsing logic mirrors mermaid grammar.
func (p *Parser) parseClassMemberLine() *ClassMember {
	member := &ClassMember{}

	// Parse visibility
	switch {
	case p.check(TokenPlus):
		member.Visibility = VisibilityPublic
		p.advance()
	case p.check(TokenMinus):
		member.Visibility = VisibilityPrivate
		p.advance()
	case p.check(TokenHash):
		member.Visibility = VisibilityProtected
		p.advance()
	case p.check(TokenTilde):
		member.Visibility = VisibilityPackage
		p.advance()
	}

	// Parse member - could be:
	// 1. "Type name" (attribute with type first)
	// 2. "name()" (method)
	// 3. "name : Type" (attribute with colon syntax)
	if p.check(TokenIdentifier) {
		first := p.current().Value
		p.advance()

		// Check for method syntax: name()
		if p.check(TokenOpenParen) {
			// Consume everything until close paren or newline
			var params strings.Builder
			params.WriteString(first)
			params.WriteString("(")
			p.advance()
			for !p.isAtEnd() && !p.check(TokenCloseParen) && !p.check(TokenNewline) {
				params.WriteString(p.current().Value)
				p.advance()
			}
			if p.check(TokenCloseParen) {
				params.WriteString(")")
				p.advance()
			}
			member.Name = params.String()

			// Optional return type: method() : Type
			if p.check(TokenColon) {
				p.advance()
				if p.check(TokenIdentifier) {
					member.Type = p.current().Value
					p.advance()
				}
			}
			return member
		}

		// Check for second identifier (Type name syntax)
		if p.check(TokenIdentifier) {
			member.Type = first
			member.Name = p.current().Value
			p.advance()
			return member
		}

		// Check for colon syntax: name : Type
		if p.check(TokenColon) {
			p.advance()
			if p.check(TokenIdentifier) {
				member.Name = first
				member.Type = p.current().Value
				p.advance()
			} else {
				member.Name = first
			}
			return member
		}

		// Just a name
		member.Name = first
		return member
	}

	return nil
}

func (p *Parser) parseClassMember(className string) error {
	class := p.findOrCreateClass(className)
	member := p.parseClassMemberLine()
	if member != nil {
		if strings.Contains(member.Name, "(") {
			class.Methods = append(class.Methods, *member)
		} else {
			class.Attributes = append(class.Attributes, *member)
		}
	}
	return nil
}

func (p *Parser) parseClassRelationship(from string) error {
	return p.parseClassRelationshipWithPrefix(from, "")
}

func (p *Parser) parseClassRelationshipWithPrefix(from string, prefix string) error {
	rel := &Relationship{From: from}
	p.findOrCreateClass(from) // Ensure the source class exists

	// Parse relationship type from arrow
	arrowStr := prefix
	for p.check(TokenArrow) || p.check(TokenLess) || p.check(TokenStar) ||
		p.check(TokenMinus) || p.check(TokenDot) || p.check(TokenPipe) ||
		p.check(TokenGreater) {
		arrowStr += p.current().Value
		p.advance()
	}

	rel.Type = parseRelationType(arrowStr)

	// Parse target class
	if p.check(TokenIdentifier) {
		rel.To = p.current().Value
		p.advance()
		p.findOrCreateClass(rel.To)
	}

	// Parse optional label
	if p.check(TokenColon) {
		p.advance()
		rel.Label = p.consumeRestOfLine()
	}

	p.diagram.Relationships = append(p.diagram.Relationships, rel)
	return nil
}

func (p *Parser) findOrCreateClass(name string) *Class {
	for _, class := range p.diagram.Classes {
		if class.Name == name {
			return class
		}
	}
	class := &Class{Name: name}
	p.diagram.Classes = append(p.diagram.Classes, class)
	return class
}

// ==== State Diagram Parsing ====

func (p *Parser) parseStateDiagram() error {
	p.diagram.Type = DiagramState
	p.advance() // consume 'stateDiagram' or 'stateDiagram-v2'
	p.skipNewlines()

	for !p.isAtEnd() {
		if err := p.parseStateStatement(); err != nil {
			return err
		}
		p.skipNewlines()
	}

	return nil
}

func (p *Parser) parseStateStatement() error {
	// Check for [*] start/end state (may be lexed as TokenLabel or TokenOpenBracket)
	if p.check(TokenLabel) && p.current().Value == "[*]" {
		p.advance() // consume [*]
		if p.check(TokenArrow) {
			return p.parseTransition("[*]")
		}
		return nil
	}

	if p.check(TokenOpenBracket) {
		return p.parseSpecialState()
	}

	if p.check(TokenIdentifier) {
		stateID := p.current().Value
		p.advance()

		// Check for transition arrow
		if p.check(TokenArrow) {
			return p.parseTransition(stateID)
		}

		// Check for state description
		if p.check(TokenColon) {
			p.advance()
			state := p.findOrCreateState(stateID)
			state.Description = p.consumeRestOfLine()
		}
	}

	if !p.isAtEnd() && !p.check(TokenNewline) {
		p.advance()
	}
	return nil
}

func (p *Parser) parseSpecialState() error {
	p.advance() // [
	if p.check(TokenStar) {
		p.advance()
	}
	if p.check(TokenCloseBracket) {
		p.advance()
	}

	// Check for transition
	if p.check(TokenArrow) {
		return p.parseTransition(stateMarker)
	}

	return nil
}

func (p *Parser) parseTransition(from string) error {
	p.advance() // consume arrow

	trans := &Transition{From: from}

	// Parse target state - could be [*] as TokenLabel or TokenOpenBracket
	switch {
	case p.check(TokenLabel) && p.current().Value == stateMarker:
		trans.To = stateMarker
		p.advance()
	case p.check(TokenOpenBracket):
		p.advance()
		if p.check(TokenStar) {
			p.advance()
		}
		if p.check(TokenCloseBracket) {
			p.advance()
		}
		trans.To = stateMarker
	case p.check(TokenIdentifier):
		trans.To = p.current().Value
		p.advance()
		p.findOrCreateState(trans.To)
	}

	// Parse transition label and guard
	if p.check(TokenColon) {
		p.advance()
		labelAndGuard := p.consumeRestOfLine()
		// Parse guard from label: "label [guard]" or just "[guard]"
		trans.Label, trans.Guard = extractGuardFromLabel(labelAndGuard)
	}

	p.diagram.Transitions = append(p.diagram.Transitions, trans)
	return nil
}

// extractGuardFromLabel separates label and guard from a transition label.
// Guards are enclosed in brackets like "event [condition]" or "[condition]".
func extractGuardFromLabel(input string) (label, guard string) {
	input = strings.TrimSpace(input)

	// Find the last occurrence of [guard] pattern
	lastOpen := strings.LastIndex(input, "[")
	lastClose := strings.LastIndex(input, "]")

	if lastOpen != -1 && lastClose > lastOpen {
		// Make sure this is not [*] for start/end state
		guardContent := input[lastOpen+1 : lastClose]
		if guardContent != "*" {
			guard = strings.TrimSpace(guardContent)
			label = strings.TrimSpace(input[:lastOpen])
			return
		}
	}

	// No guard found, entire input is the label
	label = input
	return
}

func (p *Parser) findOrCreateState(id string) *State {
	for _, state := range p.diagram.States {
		if state.ID == id {
			return state
		}
	}
	state := &State{ID: id, Label: id}
	p.diagram.States = append(p.diagram.States, state)
	return state
}

// ==== ER Diagram Parsing ====

func (p *Parser) parseERDiagram() error {
	p.diagram.Type = DiagramER
	p.advance() // consume 'erDiagram'
	p.skipNewlines()

	for !p.isAtEnd() {
		if err := p.parseERStatement(); err != nil {
			return err
		}
		p.skipNewlines()
	}

	return nil
}

func (p *Parser) parseERStatement() error {
	if !p.check(TokenIdentifier) {
		if !p.isAtEnd() {
			p.advance()
		}
		return nil
	}

	entityA := p.current().Value
	p.advance()

	// Check for relationship or entity block
	if p.check(TokenOpenBrace) {
		return p.parseEntityBlock(entityA)
	}

	// Check for relationship - ER cardinality can be arrows like "||", "o{", "}|" etc.
	if p.check(TokenPipe) || p.check(TokenOpenBrace) ||
		strings.HasPrefix(p.current().Value, "|") ||
		strings.HasPrefix(p.current().Value, "}") ||
		strings.HasPrefix(p.current().Value, "o") ||
		(p.check(TokenArrow) && isERCardinality(p.current().Value)) {
		return p.parseERRelation(entityA)
	}

	p.findOrCreateEntity(entityA)
	return nil
}

// isERCardinality checks if an arrow token is an ER cardinality indicator
func isERCardinality(s string) bool {
	erCards := []string{"||", "|o", "o|", "}|", "|{", "}o", "o{"}
	for _, card := range erCards {
		if s == card {
			return true
		}
	}
	return false
}

func (p *Parser) parseEntityBlock(name string) error {
	entity := p.findOrCreateEntity(name)
	p.advance() // consume {
	p.skipNewlines()

	for !p.isAtEnd() && !p.check(TokenCloseBrace) {
		attr := p.parseERAttribute()
		if attr != nil {
			entity.Attributes = append(entity.Attributes, *attr)
		} else if !p.isAtEnd() && !p.check(TokenCloseBrace) && !p.check(TokenNewline) {
			// Skip unknown tokens to avoid infinite loop
			p.advance()
		}
		p.skipNewlines()
	}

	if p.check(TokenCloseBrace) {
		p.advance()
	}

	return nil
}

func (p *Parser) parseERAttribute() *ERAttribute {
	attr := &ERAttribute{}

	// Type
	if p.check(TokenIdentifier) {
		attr.Type = p.current().Value
		p.advance()
	}

	// Name
	if p.check(TokenIdentifier) {
		attr.Name = p.current().Value
		p.advance()
	}

	// Key type (PK, FK, UK)
	if p.check(TokenIdentifier) {
		switch strings.ToUpper(p.current().Value) {
		case "PK":
			attr.Key = KeyPrimary
		case "FK":
			attr.Key = KeyForeign
		case "UK":
			attr.Key = KeyUnique
		}
		p.advance()
	}

	// Comment in quotes
	if p.check(TokenString) {
		attr.Comment = p.current().Value
		p.advance()
	}

	if attr.Name == "" {
		return nil
	}
	return attr
}

func (p *Parser) parseERRelation(entityA string) error {
	rel := &ERRelation{EntityA: entityA}
	p.findOrCreateEntity(entityA)

	// Parse cardinality A
	cardA := p.parseCardinality()
	rel.CardinalityA = cardA

	// Parse relationship line (may be TokenArrow "--" or ".." or individual TokenMinus/TokenDot)
	rel.Identifying = true // default to identifying (solid line)
lineScan:
	for {
		switch {
		case p.check(TokenMinus) || p.check(TokenDot):
			if p.check(TokenDot) {
				rel.Identifying = false
			}
			p.advance()
		case p.check(TokenArrow) && (p.current().Value == "--" || p.current().Value == ".."):
			if p.current().Value == ".." {
				rel.Identifying = false
			}
			p.advance()
		default:
			break lineScan
		}
	}

	// Parse cardinality B
	cardB := p.parseCardinality()
	rel.CardinalityB = cardB

	// Parse entity B
	if p.check(TokenIdentifier) {
		rel.EntityB = p.current().Value
		p.advance()
		p.findOrCreateEntity(rel.EntityB)
	}

	// Parse label
	if p.check(TokenColon) {
		p.advance()
		rel.Label = p.consumeRestOfLine()
	}

	p.diagram.ERRelations = append(p.diagram.ERRelations, rel)
	return nil
}

//nolint:gocyclo // explicit branching keeps parser straightforward.
func (p *Parser) parseCardinality() Cardinality {
	value := ""
	for p.check(TokenPipe) || p.check(TokenOpenBrace) || p.check(TokenCloseBrace) ||
		(p.check(TokenIdentifier) && (p.current().Value == "o" || p.current().Value == "O")) ||
		(p.check(TokenArrow) && (strings.Contains(p.current().Value, "|") ||
			strings.Contains(p.current().Value, "{") ||
			strings.Contains(p.current().Value, "}") ||
			strings.Contains(p.current().Value, "o"))) {
		value += p.current().Value
		p.advance()
	}

	switch {
	case strings.Contains(value, "}|") || strings.Contains(value, "|{"):
		return CardOneOrMore
	case strings.Contains(value, "}o") || strings.Contains(value, "o{"):
		return CardZeroOrMore
	case strings.Contains(value, "||"):
		return CardExactlyOne
	case strings.Contains(value, "|o") || strings.Contains(value, "o|"):
		return CardZeroOrOne
	default:
		return CardExactlyOne
	}
}

func (p *Parser) findOrCreateEntity(name string) *Entity {
	for _, entity := range p.diagram.Entities {
		if entity.Name == name {
			return entity
		}
	}
	entity := &Entity{Name: name}
	p.diagram.Entities = append(p.diagram.Entities, entity)
	return entity
}

// ==== Pie Chart Parsing ====

func (p *Parser) parsePieDiagram() error {
	p.diagram.Type = DiagramPie
	p.advance() // consume 'pie'
	p.skipNewlines()

	// Parse optional directives and title
	for !p.isAtEnd() {
		// Check for showData directive
		if p.check(TokenKeyword) && p.current().Value == "showData" {
			p.diagram.ShowData = true
			p.advance()
			p.skipNewlines()
			continue
		}

		// Check for title
		if p.check(TokenKeyword) && p.current().Value == "title" {
			p.advance()
			p.diagram.Title = p.consumeRestOfLine()
			p.skipNewlines()
			continue
		}

		// Parse pie slice: "Label" : value
		if p.check(TokenString) {
			if err := p.parsePieSlice(); err != nil {
				return err
			}
			p.skipNewlines()
			continue
		}

		// Skip unknown tokens
		if !p.isAtEnd() {
			p.advance()
		}
		p.skipNewlines()
	}

	return nil
}

func (p *Parser) parsePieSlice() error {
	slice := &PieSlice{}

	// Parse label (string in quotes)
	if p.check(TokenString) {
		slice.Label = p.current().Value
		p.advance()
	}

	// Expect colon
	if p.check(TokenColon) {
		p.advance()
	}

	// Parse value (number)
	if p.check(TokenNumber) {
		value, err := parseFloat(p.current().Value)
		if err != nil {
			return p.error("invalid pie slice value")
		}
		slice.Value = value
		p.advance()
	}

	if slice.Label != "" && slice.Value > 0 {
		p.diagram.PieSlices = append(p.diagram.PieSlices, slice)
	}

	return nil
}

// parseFloat parses a float from a string.
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// ==== Helper Methods ====

func (p *Parser) check(tt TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.current().Type == tt
}

// isFlowchartIdentifier returns true if the current token can be used as a node identifier
// in a flowchart. This includes regular identifiers and keywords that are valid as node names
// (e.g., "loop", "note", "class", "end" etc. which are keywords in other diagram types).
func (p *Parser) isFlowchartIdentifier() bool {
	if p.isAtEnd() {
		return false
	}
	t := p.current().Type
	// Regular identifier
	if t == TokenIdentifier {
		return true
	}
	// Keywords that can be used as node names in flowcharts
	// Many mermaid keywords are valid node IDs in flowchart context
	switch t {
	case TokenLoop, TokenNote, TokenAlt, TokenOpt, TokenPar, TokenRect,
		TokenParticipant, TokenActor, TokenClass, TokenEnd, TokenSubgraph,
		TokenKeyword, TokenDirection, TokenElse:
		return true
	}
	return false
}

func (p *Parser) current() Token {
	if p.isAtEnd() {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() Token {
	if !p.isAtEnd() {
		p.pos++
	}
	return p.tokens[p.pos-1]
}

func (p *Parser) isAtEnd() bool {
	return p.pos >= len(p.tokens) || p.tokens[p.pos].Type == TokenEOF
}

func (p *Parser) skipNewlines() {
	for p.check(TokenNewline) {
		p.advance()
	}
}

func (p *Parser) consumeRestOfLine() string {
	var parts []string
	for !p.isAtEnd() && !p.check(TokenNewline) {
		parts = append(parts, p.current().Value)
		p.advance()
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// consumeUntilPipe consumes tokens until a pipe is found, supporting multi-word edge labels.
func (p *Parser) consumeUntilPipe() string {
	var parts []string
	for !p.isAtEnd() && !p.check(TokenPipe) && !p.check(TokenNewline) {
		parts = append(parts, p.current().Value)
		p.advance()
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func (p *Parser) error(msg string) error {
	tok := p.current()
	return &ParseError{
		Message: msg,
		Line:    tok.Line,
		Column:  tok.Column,
	}
}

// ==== Utility Functions ====

func extractLabel(value string) string {
	// Remove outer delimiters using longest-match-first approach
	// This matches mermaid-cli behavior for shape delimiter pairs
	v := strings.TrimSpace(value)
	if len(v) < 2 {
		return v
	}

	// Delimiter pairs ordered by longest opener first to prevent premature stripping
	// e.g., "([Stadium])" must match "([ ])" not "( )" with leftover brackets
	type pair struct{ open, close string }
	pairs := []pair{
		{"(((", ")))"}, // double circle
		{"((", "))"},   // circle
		{"([", "])"},   // stadium
		{"[(", ")]"},   // cylinder
		{"[[", "]]"},   // subroutine
		{"{{", "}}"},   // hexagon
		{"[/", "\\]"},  // trapezoid
		{"[\\", "/]"},  // trapezoid alt
		{"[/", "/]"},   // parallelogram
		{"[\\", "\\]"}, // parallelogram alt
		{">", "]"},     // asymmetric
		{"{", "}"},     // diamond
		{"[", "]"},     // rectangle
		{"(", ")"},     // rounded
	}

	for _, p := range pairs {
		if strings.HasPrefix(v, p.open) && strings.HasSuffix(v, p.close) {
			return strings.TrimSpace(v[len(p.open) : len(v)-len(p.close)])
		}
	}

	return v
}

//nolint:gocyclo // explicit branching keeps parser straightforward.
func detectShape(value string) NodeShape {
	v := strings.TrimSpace(value)

	switch {
	case strings.HasPrefix(v, "(((") && strings.HasSuffix(v, ")))"):
		return ShapeDoubleCircle
	case strings.HasPrefix(v, "((") && strings.HasSuffix(v, "))"):
		return ShapeCircle
	case strings.HasPrefix(v, "([") && strings.HasSuffix(v, "])"):
		return ShapeStadium
	case strings.HasPrefix(v, "[[") && strings.HasSuffix(v, "]]"):
		return ShapeSubroutine
	case strings.HasPrefix(v, "[(") && strings.HasSuffix(v, ")]"):
		return ShapeCylinder
	case strings.HasPrefix(v, "{{") && strings.HasSuffix(v, "}}"):
		return ShapeHexagon
	case strings.HasPrefix(v, "{") && strings.HasSuffix(v, "}"):
		return ShapeRhombus
	case strings.HasPrefix(v, "(") && strings.HasSuffix(v, ")"):
		return ShapeRounded
	case strings.HasPrefix(v, "[/") && strings.HasSuffix(v, "/]"):
		return ShapeParallelogram
	case strings.HasPrefix(v, "[\\") && strings.HasSuffix(v, "\\]"):
		return ShapeParallelogramAlt
	case strings.HasPrefix(v, "[/") && strings.HasSuffix(v, "\\]"):
		return ShapeTrapezoid
	case strings.HasPrefix(v, "[\\") && strings.HasSuffix(v, "/]"):
		return ShapeTrapezoidAlt
	case strings.HasPrefix(v, ">") && strings.HasSuffix(v, "]"):
		return ShapeAsymmetric
	case strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]"):
		return ShapeRectangle
	default:
		return ShapeRectangle
	}
}

func parseArrowStyle(arrow string) (EdgeType, ArrowType, ArrowType) {
	edgeType := EdgeSolid
	arrowStart := ArrowNone
	arrowEnd := ArrowNone

	switch {
	case strings.Contains(arrow, "===") || strings.HasPrefix(arrow, "=="):
		edgeType = EdgeThick
	case strings.Contains(arrow, "-.-") || strings.Contains(arrow, "-."):
		edgeType = EdgeDotted
	case strings.Contains(arrow, "~~~"):
		edgeType = EdgeInvisible
	}

	if strings.HasPrefix(arrow, "<") {
		arrowStart = ArrowNormal
	}
	if strings.HasPrefix(arrow, "o") {
		arrowStart = ArrowCircle
	}
	if strings.HasPrefix(arrow, "x") {
		arrowStart = ArrowCross
	}

	switch {
	case strings.HasSuffix(arrow, ">"):
		arrowEnd = ArrowNormal
	case strings.HasSuffix(arrow, "o"):
		arrowEnd = ArrowCircle
	case strings.HasSuffix(arrow, "x"):
		arrowEnd = ArrowCross
	}

	return edgeType, arrowStart, arrowEnd
}

func parseMessageType(arrow string) MessageType {
	switch arrow {
	case "->>":
		return MessageSolidArrow
	case "-->>":
		return MessageDottedArrow
	case "-x", "-X":
		return MessageSolidCross
	case "--x", "--X":
		return MessageDottedCross
	case "-)":
		return MessageSolidOpen
	case "--)":
		return MessageDottedOpen
	case "-->":
		return MessageDotted
	default:
		return MessageSolid
	}
}

func parseRelationType(arrow string) RelationType {
	switch {
	case strings.Contains(arrow, "<|--"):
		return RelationInheritance
	case strings.Contains(arrow, "<|..") || strings.Contains(arrow, "..|>"):
		return RelationRealization
	case strings.Contains(arrow, "*--"):
		return RelationComposition
	case strings.Contains(arrow, "o--"):
		return RelationAggregation
	case strings.Contains(arrow, "..>"):
		return RelationDependency
	default:
		return RelationAssociation
	}
}
