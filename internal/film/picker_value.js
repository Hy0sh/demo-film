// The value a native input of that type keeps for the value given, as the
// browser normalises it (a colour in lower case, zero seconds dropped from a
// time), or "" when the input refuses it.
([type, value]) => {
  const input = document.createElement("input");
  input.type = type;
  input.value = value;
  return input.value;
}
