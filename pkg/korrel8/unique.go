package korrel8

type uniqueObjects map[Identifier]Object

func (u uniqueObjects) add(objs []Object) {
	for _, o := range objs {
		u[o.Identifier()] = o
	}
}

func (u uniqueObjects) list() (objs []Object) {
	for _, o := range u {
		objs = append(objs, o)
	}
	return objs
}

func uniqueObjectList(objs []Object) []Object {
	u := uniqueObjects{}
	u.add(objs)
	return u.list()
}

type unique[T comparable] map[T]struct{}

func (u unique[T]) add(values []T) {
	for _, v := range values {
		u[v] = struct{}{}
	}
}

func (u unique[T]) list() (values []T) {
	for v := range u {
		values = append(values, v)
	}
	return values
}
