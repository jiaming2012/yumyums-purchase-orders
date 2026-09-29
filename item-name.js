// item-name.js — the one answer to "what do we call this item?"
//
// WHY THIS EXISTS
// ---------------
// An inventory item has up to three names and they are not interchangeable:
//
//   description   the catalog name. IDENTITY — it keys stock_count_overrides,
//                 the expanded-row maps, the "View in Setup" jump, and it
//                 wins every receipt auto-match. Often unreadable, because
//                 the receipt worker names auto-created items with the raw
//                 receipt string ("100% Cl Hny 24Z Bram").
//   aliases[]     a BAG of alternate names. Mostly machine-learned receipt
//                 text; sometimes a human nickname. You cannot tell which is
//                 which by looking at the string.
//   display name  the ONE alias a human promoted (item_aliases.is_display),
//                 falling back to description. Resolved server-side by the
//                 SQL function item_display_name() so every endpoint agrees.
//
// Before this module, "show the nickname" was attempted once, locally, on the
// Purchases card: take the first alias that is neither the receipt text nor
// the catalog name. That only worked there because that screen happens to
// hold the receipt text to exclude. Lifted anywhere else it renders
// "100% CL HNY 24Z BRAM" — the auto-learned string — about as often as the
// readable one. Hence a promoted flag, and hence this module.
//
// THE RULE
// --------
//   itemName(o)      → what to SHOW. Never blank when any name exists.
//   itemSubName(o)   → the catalog name, but ONLY when it differs from what
//                      itemName returned. '' otherwise, so callers can
//                      concatenate without testing for duplicates.
//   itemIdentity(o)  → the catalog name, always. Use for keys, data-* attrs
//                      and anything sent back to the server. NEVER render it
//                      as the primary label; never key off itemName().
(function (global) {
  'use strict';

  function firstString() {
    for (var i = 0; i < arguments.length; i++) {
      var v = arguments[i];
      if (typeof v === 'string' && v.trim() !== '') return v;
    }
    return '';
  }

  // Server-resolved display name, under the two json tags it ships as:
  // `display_name` on items / stock / suggestions / shopping rows, and
  // `item_display_name` on a purchase line (where `description` is already
  // taken by the receipt's own text).
  function itemName(o) {
    if (!o) return '';
    return firstString(o.display_name, o.item_display_name, itemIdentity(o));
  }

  // The catalog name. `item_name` before `description` on purpose: on a line
  // item, `description` is the RECEIPT's text and `item_name` is the catalog
  // entry it was linked to. On every other shape only one of them exists.
  function itemIdentity(o) {
    if (!o) return '';
    return firstString(o.item_name, o.description, o.name);
  }

  function itemSubName(o) {
    var canonical = itemIdentity(o);
    var shown = itemName(o);
    if (!canonical || canonical.toLowerCase() === shown.toLowerCase()) return '';
    return canonical;
  }

  global.itemName = itemName;
  global.itemSubName = itemSubName;
  global.itemIdentity = itemIdentity;
})(this);
