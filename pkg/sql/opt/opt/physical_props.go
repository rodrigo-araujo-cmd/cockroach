// Copyright 2018 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package opt

import (
	"bytes"
	"fmt"
)

// PhysicalPropsID identifies a set of physical properties that has been
// interned by a memo instance. If two ids are the same, then the physical
// properties are the same.
type PhysicalPropsID uint32

const (
	// NormPhysPropsID is a special id used to create and traverse a normalized
	// expression tree before it's been fully explored and costed. Before then,
	// its physical properties are still undefined, and they cannot not be
	// used. But the id itself can still be used with ExprView to traverse the
	// normalized logical expression tree (with no enforcers or physical
	// properties present).
	//
	// While NormPhysPropsID and MinPhysPropsID refer to the same default
	// normalized expression at the beginning of optimization, the
	// MinPhysPropsID expression tree quickly diverges as the optimizer finds
	// alternate expressions with lower cost.
	NormPhysPropsID PhysicalPropsID = 1

	// MinPhysPropsID is the id of the well-known set of physical properties
	// that requires nothing of an operator. Therefore, every operator is
	// guaranteed to provide this set of properties. This is typically the most
	// commonly used set of physical properties in the memo, since most
	// operators do not require any physical properties from their children.
	MinPhysPropsID PhysicalPropsID = 2
)

// PhysicalProps are interesting characteristics of an expression that impact
// its layout, presentation, or location, but not its logical content. Examples
// include row order, column naming, and data distribution (physical location
// of data ranges). Physical properties exist outside of the relational
// algebra, and arise from both the SQL query itself (e.g. the non-relational
// ORDER BY operator) and by the selection of specific implementations during
// optimization (e.g. a merge join requires the inputs to be sorted in a
// particular order).
//
// Physical properties can be provided by an operator or required of it. Some
// operators "naturally" provide a physical property such as ordering on a
// particular column. Other operators require one or more of their operands to
// provide a particular physical property. When an expression is optimized, it
// is always with respect to a particular set of required physical properties.
// The goal is to find the lowest cost expression that provides those
// properties while still remaining logically equivalent.
type PhysicalProps struct {
	// Presentation specifies the naming, membership (including duplicates),
	// and order of result columns. If Presentation is not defined, then no
	// particular column presentation is required or provided.
	Presentation Presentation
}

// Defined returns true if any physical property is defined. If none is
// defined, then this is an instance of MinPhysProps.
func (p *PhysicalProps) Defined() bool {
	return p.Presentation.Defined()
}

// Fingerprint returns a string that uniquely describes this set of physical
// properties. It is suitable for use as a hash key in a map.
func (p *PhysicalProps) Fingerprint() string {
	hasProjection := p.Presentation.Defined()

	// Handle default properties case.
	if !hasProjection {
		return ""
	}

	var buf bytes.Buffer

	if hasProjection {
		buf.WriteString("p:")
		p.Presentation.format(&buf)
	}

	return buf.String()
}

// Presentation specifies the naming, membership (including duplicates), and
// order of result columns that are required of or provided by an operator.
// While it cannot add unique columns, Presentation can rename, reorder,
// duplicate and discard columns. If Presentation is not defined, then no
// particular column presentation is required or provided. For example:
//   a.y:2 a.x:1 a.y:2 column1:3
type Presentation []LabeledColumn

// Defined is true if a particular column presentation is required or provided.
func (p Presentation) Defined() bool {
	return p != nil
}

// Provides returns true iff this presentation exactly matches the given
// presentation.
func (p Presentation) Provides(required Presentation) bool {
	if len(p) != len(required) {
		return false
	}

	for i := 0; i < len(p); i++ {
		if p[i] != required[i] {
			return false
		}
	}
	return true
}

func (p Presentation) String() string {
	var buf bytes.Buffer
	p.format(&buf)
	return buf.String()
}

func (p Presentation) format(buf *bytes.Buffer) {
	for i, col := range p {
		if i > 0 {
			buf.WriteString(",")
		}

		fmt.Fprintf(buf, "%s:%d", col.Label, col.Index)
	}
}

// LabeledColumn specifies the label and index of a column.
type LabeledColumn struct {
	Label string
	Index ColumnIndex
}
