package main

// Components are plain data. The getter lets a system ask for them.

type Heat struct{ Degrees float32 }

func (h *Heat) GetHeat() *Heat { return h }

type Name struct{ Value string }

func (n *Name) GetName() *Name { return n }
