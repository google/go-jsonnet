package ast

func (i *IdentifierSet) Append(idents Identifiers) {
	for _, ident := range idents {
		i.Add(ident)
	}
}
