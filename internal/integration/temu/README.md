# TEMU Integration

Owns TEMU external-system adapters that are not listing-target business rules.

`OfficialClient` uses the fixed official routers, bounded standard HTTP transport,
exact UTF-8 JSON signing and private credential variant admitted by #31.
Identity verification remains separate from publishing or merchant eligibility.

The named category reads implement E33/E34. Supplier categories remain on CN:
current CN token expiry and the exact category scope are queried on CN before
metadata is read. PA permissions cannot authorize an unmigrated CN interface.
Local categories use the original US/EU/GLOBAL application route and only a
verified LOCAL mall. Both reads share one total deadline with their scope query.
The two response types retain their different flags; a missing list is an error,
an explicit empty list is an observation, and missing availability is unknown.

These are internal server adapters. The current Store product handle and shared
TargetRecord consumers still need their typed TEMU variants; there is no public
arbitrary-API proxy, active merchant credential, publishing or acceptance claim.
