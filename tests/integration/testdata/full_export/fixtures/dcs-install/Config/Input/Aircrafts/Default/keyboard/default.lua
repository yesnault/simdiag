-- Trimmed stand-in for DCS World's own
-- Config/Input/Aircrafts/Default/keyboard/default.lua.
--
-- Only the F1-F12 view commands are kept: parseDefaultKeyboardBindings ignores
-- every other key, and these are the labels the export attaches to a Gremlins
-- binding that maps a device button onto a function key.
--
-- The labels are the ones expected.csv already carried. They used to come from
-- whatever DCS happened to be installed on the machine running the test, which
-- is why CI, where DCS is not installed, produced rows with an empty action.
-- F8 is deliberately absent: no default view command is bound to it.

return {
	keyCommands = {
		{combos = {{key = 'F1'}}, down = iCommandViewCockpit, name = _('F1 Cockpit view'), category = _('View')},
		{combos = {{key = 'F2'}}, down = iCommandViewAircraft, name = _('F2 Aircraft view'), category = _('View')},
		{combos = {{key = 'F3'}}, down = iCommandViewFlyBy, name = _('F3 Fly-By view'), category = _('View')},
		{combos = {{key = 'F4'}}, down = iCommandViewChase, name = _('F4 Camera mounted on object'), category = _('View')},
		{combos = {{key = 'F5'}}, down = iCommandViewNearestAC, name = _('F5 nearest AC view'), category = _('View')},
		{combos = {{key = 'F6'}}, down = iCommandViewWeapon, name = _('F6 Released weapon view'), category = _('View')},
		{combos = {{key = 'F7'}}, down = iCommandViewGroundUnit, name = _('F7 Ground unit view'), category = _('View')},
		{combos = {{key = 'F9'}}, down = iCommandViewShip, name = _('F9 Ship view'), category = _('View')},
		{combos = {{key = 'F10'}}, down = iCommandViewTheatreMap, name = _('F10 Theater map view'), category = _('View')},
		{combos = {{key = 'F11'}}, down = iCommandViewFreeAirport, name = _('F11 Airport free camera'), category = _('View')},
		{combos = {{key = 'F12'}}, down = iCommandViewStaticObject, name = _('F12 Static object view'), category = _('View')},
	},
}
