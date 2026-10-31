"""Fixed commands; ERPNext owns all quantities and commits the receipt with them.

No provider call, commit, schema installation or arbitrary document proxy belongs here.
The authenticated application rechecks current member/Store authority on every call.
"""
import hashlib
import json
import math
import uuid
from datetime import timezone
from decimal import Decimal, InvalidOperation
from zoneinfo import ZoneInfo

import frappe
from frappe.utils import add_to_date, get_datetime, get_system_timezone, now_datetime
from frappe.utils.password import encrypt, decrypt
from erpnext.buying.doctype.purchase_order.purchase_order import make_purchase_receipt
from erpnext.selling.doctype.sales_order.sales_order import make_delivery_note
from erpnext.stock.doctype.stock_reservation_entry.stock_reservation_entry import get_available_qty_to_reserve

from commerce_fulfillment.install import ROLE

OP = "Commerce Fulfillment Operation"


def fail():
    frappe.throw("Fulfillment input, binding or current state conflicts")


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False, allow_nan=False)


def digest(value):
    return hashlib.sha256(canonical(value).encode()).hexdigest()


def text(value, maximum=140):
    if not isinstance(value, str) or not value.strip() or len(value) > maximum or any(ord(c) < 32 for c in value):
        fail()
    return value


def shape(value, required, optional=()):
    if not isinstance(value, dict) or set(value) - set(required) - set(optional) or set(required) - set(value):
        fail()


def quantity(value):
    if isinstance(value, bool):
        fail()
    try:
        number = Decimal(str(value))
        if not number.is_finite() or number <= 0 or number != number.to_integral_value() or number > 100000000:
            fail()
    except (InvalidOperation, ValueError, TypeError):
        fail()
    return int(number)


def lines(payload, fields):
    rows = payload.get("lines")
    if not isinstance(rows, list) or not 1 <= len(rows) <= 20:
        fail()
    seen = set()
    identity = fields[0]
    for row in rows:
        shape(row, fields)
        key = text(row[identity])
        if key in seen:
            fail()
        seen.add(key)
        quantity(row["qty"])
    return rows


def lock_doc(doctype, name, cfg, draft=False):
    # Only literals selected by this module reach doctype; never an HTTP argument.
    frappe.db.sql("select name from `tab" + doctype + "` where name=%s for update", (text(name),))
    doc = frappe.get_doc(doctype, name, for_update=True)
    if doc.company != cfg["company"] or doc.docstatus not in ((0, 1) if draft else (1,)):
        fail()
    return doc


def demand_doc(name, request, cfg):
    doc = lock_doc("Sales Order", name, cfg)
    if not doc.cf_source or doc.cf_binding != request["binding"] or doc.status in ("Closed", "Cancelled"):
        fail()
    source = json.loads(doc.cf_source)
    if source["organization"] != request["organization"] or any(source[key] != request[key] for key in ("store", "source_kind", "source_id")):
        fail()
    current = request.get("current_lines")
    if current is not None:
        remaining = {row.cf_source_line: int(row.qty - row.delivered_qty) for row in doc.items if row.qty > row.delivered_qty}
        if not isinstance(current, dict) or current != remaining:
            fail()
    return doc


def selected(doc, rows, field):
    by_name = {row.name: row for row in doc.items}
    if any(row[field] not in by_name for row in rows):
        fail()
    return [(by_name[row[field]], quantity(row["qty"])) for row in rows]


def allocation_budget(line):
    purchases = frappe.db.sql("select item.name,item.qty,item.received_qty,po.status from `tabPurchase Order Item` item join `tabPurchase Order` po on po.name=item.parent where item.sales_order_item=%s and po.docstatus=1 for update", (line.name,), as_dict=True)
    committed, ordered = 0, 0
    for purchase in purchases:
        rejected = frappe.db.sql("select coalesce(sum(rejected_qty),0) from `tabPurchase Receipt Item` where purchase_order_item=%s and docstatus=1 for update", (purchase.name,))[0][0]
        ordered += purchase.qty
        committed += (purchase.received_qty if purchase.status == "Closed" else purchase.qty) - rejected
    remaining, existing = frappe.db.sql("select coalesce(sum(reserved_qty-delivered_qty),0),coalesce(sum(case when coalesce(from_voucher_type,'')!='Purchase Receipt' then reserved_qty else 0 end),0) from `tabStock Reservation Entry` where voucher_detail_no=%s and docstatus=1 for update", (line.name,))[0]
    reserved = remaining / line.conversion_factor
    # Receipt-origin reservations overlap their own PO commitment. Existing
    # stock reservations do not. Keep that provenance in native SRE, not a ledger.
    free = max(0, line.qty - committed - existing / line.conversion_factor)
    return {"ordered_qty": float(ordered), "reserved_qty": float(reserved), "reserve_budget": math.floor(free),
        "purchase_remaining": math.floor(free),
        "reservable_qty": math.floor(max(0, min(free, get_available_qty_to_reserve(line.item_code, line.warehouse)/line.conversion_factor)))}


def receipt(doc):
    result = json.loads(doc.result_json or "{}")
    return {"operation": doc.name, "stage": doc.stage, "target": doc.target, "revision": doc.revision, "result": result}


@frappe.whitelist(methods=["POST"])
def execute(request):
    # Frappe's default exception logger may include form arguments and local values.
    # This RPC never exposes raw request data or unexpected traceback context.
    frappe.local.flags.disable_traceback = True
    if getattr(frappe.local, "form_dict", None):
        frappe.local.form_dict.pop("request", None)
    savepoint = "cf_" + uuid.uuid4().hex
    frappe.db.savepoint(savepoint)
    try:
        return _execute(request)
    except Exception as error:
        frappe.db.rollback(save_point=savepoint)
        if isinstance(error, (frappe.ValidationError, frappe.PermissionError)):
            raise
        frappe.throw("Fulfillment outcome requires original-operation verification", frappe.ValidationError)


def _execute(request):
    if isinstance(request, str):
        if len(request.encode()) > 256 * 1024:
            fail()
        request = json.loads(request)
    shape(request, ("operation_id", "organization", "actor", "member", "binding", "store", "source_kind", "source_id", "command", "payload"), ("current_lines",))
    if len(canonical(request).encode()) > 256 * 1024:
        fail()
    cfg = frappe.conf.get("commerce_fulfillment")
    test_site = bool(frappe.conf.get("commerce_fulfillment_test_site"))
    if not cfg or frappe.session.user != cfg.get("service_user") or (frappe.session.user == "Administrator" and not test_site):
        frappe.throw("Fulfillment service is unavailable", frappe.PermissionError)
    if not test_site:
        frappe.only_for(ROLE)
    if request["organization"] != cfg.get("organization"):
        frappe.throw("Wrong organization", frappe.PermissionError)
    if frappe.db.get_value("Company", cfg["company"], "enable_perpetual_inventory"):
        frappe.throw("Fulfillment requires the configured periodic-inventory company")
    if request["command"] == "configuration":
        if not frappe.db.get_single_value("Stock Settings", "enable_stock_reservation") or not frappe.db.get_single_value("Stock Settings", "allow_partial_reservation"):
            fail()
        return {"organization": cfg["organization"], "version": "0.1.0", "supply_movement": cfg.get("supply_movement"),
            "address_enabled": bool(frappe.conf.get("encryption_key") and isinstance(cfg.get("address_retention_seconds"), int) and 1 <= cfg["address_retention_seconds"] <= 86400)}
    if request["command"] == "load_operation":
        shape(request["payload"], ("operation",))
        doc = frappe.get_doc(OP, text(request["payload"]["operation"]))
        if doc.organization != cfg["organization"] or doc.purpose != "prepare":
            fail()
        demand = frappe.get_doc("Sales Order", doc.target)
        if demand.company != cfg["company"]:
            fail()
        return {**receipt(doc), "actor": doc.actor, "member": doc.member, "binding": doc.binding,
            "input": json.loads(doc.input_json), "source": json.loads(demand.cf_source)}
    for key in ("organization", "actor", "member", "store", "source_kind", "source_id", "command"):
        text(request[key])
    if len(text(request["binding"])) != 64:
        fail()
    try:
        uuid.UUID(request["operation_id"])
    except (ValueError, TypeError, AttributeError):
        fail()
    command = request["command"]
    payload = request["payload"]
    if command == "cancel_original":
        shape(payload, ())
        # Unique insertion serializes with any late/in-flight original native
        # transaction. Never infer absence from an ordinary failed read.
        tombstone = frappe.get_doc({"doctype": OP, "operation_id": request["operation_id"],
            "organization": request["organization"], "actor": request["actor"], "member": request["member"],
            "binding": request["binding"], "purpose": command, "input_digest": digest(request),
            "stage": "DEFINITIVE_NO_EFFECT", "input_json": "{}",
            "result_json": canonical({"reason": "original_command_cancelled_before_commit"})})
        try:
            tombstone.insert()
            return receipt(tombstone)
        except frappe.DuplicateEntryError:
            original = current_operation({"operation": request["operation_id"]}, request)
            if original.target:
                validate_operation_target(original, request, cfg)
            if original.purpose == "prepare" and original.stage in ("PREPARED", "CONFIRMED"):
                return transition("cancel", {"operation": original.name, "version": original.revision}, request, cfg)
            return receipt(original)
    if command in ("read", "read_private", "find_demand", "details", "delivery_details", "catalog", "list"):
        return read_command(command, payload, request, cfg)
    if command in ("confirm", "cancel", "claim", "mark_dispatching", "record_result"):
        return transition(command, payload, request, cfg)
    handler = HANDLERS.get(command)
    if not handler:
        fail()
    input_digest = digest(request)
    # Unique insert serializes concurrent original-key retries. No check-then-write.
    operation = frappe.get_doc({"doctype": OP, "operation_id": request["operation_id"],
        "organization": request["organization"], "actor": request["actor"], "member": request["member"],
        "binding": request["binding"], "purpose": command, "input_digest": input_digest,
        "stage": "PREPARED" if command == "prepare" else "PROCESSING", "input_json": canonical(payload)})
    try:
        operation.insert()
    except frappe.DuplicateEntryError:
        existing = frappe.get_doc(OP, request["operation_id"], for_update=True)
        if existing.input_digest != input_digest:
            fail()
        return receipt(existing)
    savepoint = "cf_native_" + uuid.uuid4().hex
    frappe.db.savepoint(savepoint)
    try:
        return apply_operation(operation, handler, payload, request, cfg, input_digest, command)
    except frappe.ValidationError:
        # A known native rejection has no business effect: rollback the entire
        # native command, but retain its original immutable receipt atomically.
        # Unknown exceptions still roll back the whole RPC and remain UNKNOWN.
        frappe.db.rollback(save_point=savepoint)
        operation.reload()
        operation.stage = "DEFINITIVE_NO_EFFECT"
        operation.result_json = canonical({"reason": "native_validation_rejected"})
        operation.save()
        return receipt(operation)


def apply_operation(operation, handler, payload, request, cfg, input_digest, command):
    target, result, effect = handler(payload, request, cfg)
    if effect:
        # A different UUID cannot unlock a claimed/unknown physical/platform effect.
        operation.effect_key = digest([request["organization"], effect])
        if frappe.db.exists(OP, {"effect_key": operation.effect_key}):
            fail()
    operation.target, operation.result_json = target, canonical(result)
    operation.revision = payload.get("version") if command == "prepare" else input_digest
    operation.stage = "PREPARED" if command == "prepare" else "SUCCEEDED"
    if command == "prepare" and payload["purpose"] == "consumer_tracking":
        note = lock_doc("Delivery Note", payload["delivery"], cfg, draft=True)
        demand = demand_doc(payload["demand"], request, cfg)
        source_lines = {row.name: row.cf_source_line for row in demand.items}
        for row in note.items:
            source_line = source_lines[row.so_detail]
            operation.append("effects", {"effect_key": digest([request["organization"], request["store"], request["source_kind"], request["source_id"], "consumer_tracking", source_line]), "source_line": source_line})
    operation.save()
    return receipt(operation)


def bind_demand(p, r, c):
    shape(p, ("source_kind", "store", "source_id", "movement_kind", "delivery_date", "lines"))
    if p["source_kind"] not in ("consumer_order", "platform_supply"):
        fail()
    source = {key: text(p[key]) for key in ("source_kind", "store", "source_id")}
    if any(source[key] != r[key] for key in source):
        fail()
    source["organization"] = r["organization"]
    movement = p["movement_kind"]
    expected = "commercial_sale" if p["source_kind"] == "consumer_order" else c.get("supply_movement")
    if movement != expected or movement not in ("commercial_sale", "owned_transfer"):
        fail()
    customer = c["internal_customer"] if movement == "owned_transfer" else c["customer"]
    if movement == "owned_transfer":
        native_customer = frappe.get_doc("Customer", customer)
        if not native_customer.is_internal_customer or native_customer.represents_company != c["company"]:
            fail()
    currency = frappe.db.get_value("Company", c["company"], "default_currency")
    items = []
    for row in lines(p, ("source_line", "item_code", "qty", "uom", "warehouse")):
        item = frappe.get_doc("Item", text(row["item_code"]))
        if item.disabled or not item.is_stock_item or row["warehouse"] != c["source_warehouse"] or row["warehouse"] == c["transit_warehouse"]:
            fail()
        # Native conversion is authoritative. Explicit configured mapping only.
        factors = {item.stock_uom: 1, **{u.uom: u.conversion_factor for u in item.uoms}}
        if row["uom"] not in factors or not factors[row["uom"]] or factors[row["uom"]] <= 0:
            fail()
        items.append({"item_code": item.name, "qty": quantity(row["qty"]), "uom": row["uom"],
            "conversion_factor": factors[row["uom"]], "rate": 0, "warehouse": row["warehouse"],
            "delivery_date": p["delivery_date"], "reserve_stock": 1, "cf_source_line": row["source_line"]})
    doc = frappe.get_doc({"doctype": "Sales Order", "company": c["company"], "customer": customer,
        "delivery_date": p["delivery_date"], "currency": currency, "conversion_rate": 1,
        "selling_price_list": c["selling_price_list"], "price_list_currency": currency, "plc_conversion_rate": 1,
        "cf_source": canonical(source), "cf_binding": r["binding"], "cf_movement": movement, "items": items}).insert().submit()
    return doc.name, {}, ["demand", source]


def create_purchase(p, r, c):
    shape(p, ("demand", "supplier", "delivery_date", "lines"))
    doc = demand_doc(p["demand"], r, c)
    rows = lines(p, ("demand_line", "qty", "rate"))
    items = []
    for row in rows:
        line, qty = selected(doc, [row], "demand_line")[0]
        if qty > allocation_budget(line)["purchase_remaining"] or isinstance(row["rate"], bool):
            fail()
        try:
            rate = Decimal(str(row["rate"]))
            if not rate.is_finite() or rate < 0 or rate > 100000000:
                fail()
        except (InvalidOperation, ValueError):
            fail()
        items.append({"item_code": line.item_code, "qty": qty, "uom": line.uom, "conversion_factor": line.conversion_factor,
            "rate": float(rate), "warehouse": line.warehouse, "schedule_date": p["delivery_date"],
            "sales_order": doc.name, "sales_order_item": line.name})
    currency = frappe.db.get_value("Company", c["company"], "default_currency")
    purchase = frappe.get_doc({"doctype": "Purchase Order", "company": c["company"], "supplier": text(p["supplier"]),
        "currency": currency, "conversion_rate": 1, "buying_price_list": c["buying_price_list"],
        "price_list_currency": currency, "plc_conversion_rate": 1, "schedule_date": p["delivery_date"], "items": items}).insert().submit()
    return purchase.name, {}, None


def purchase_doc(name, r, c, allow_closed=False):
    purchase = lock_doc("Purchase Order", name, c)
    for row in purchase.items:
        demand_doc(row.sales_order, r, c)
    if purchase.status == "Cancelled" or purchase.status == "Closed" and not allow_closed:
        fail()
    return purchase


def record_external_order(p, r, c):
    shape(p, ("purchase", "external_order"))
    doc = purchase_doc(p["purchase"], r, c)
    external = text(p["external_order"])
    if doc.cf_external_order and doc.cf_external_order != external:
        fail()
    doc.cf_external_order = external
    doc.save()
    return doc.name, {}, None


def close_purchase(p, r, c):
    shape(p, ("purchase", "reason"))
    doc = purchase_doc(p["purchase"], r, c)
    text(p["reason"], 500)
    doc.check_permission("write")
    doc.update_status("Closed")
    # Reason remains on the immutable original operation; no supplier mutation.
    return doc.name, {}, None


def receive(p, r, c):
    shape(p, ("purchase", "lines"))
    doc = purchase_doc(p["purchase"], r, c)
    raw = p.get("lines")
    if not isinstance(raw, list) or not 1 <= len(raw) <= 20:
        fail()
    rows, seen = [], set()
    for entry in raw:
        shape(entry, ("purchase_line", "qty"), ("rejected_qty", "short_qty"))
        if entry["purchase_line"] in seen:
            fail()
        seen.add(entry["purchase_line"])
        def nonnegative(value):
            return 0 if value == 0 and not isinstance(value, bool) else quantity(value)
        accepted, rejected, short = [nonnegative(entry.get(key, 0)) for key in ("qty", "rejected_qty", "short_qty")]
        if not accepted + rejected:
            fail()
        row = next((item for item in doc.items if item.name == entry["purchase_line"]), None)
        if not row:
            fail()
        rows.append((row, accepted, rejected, short))
    quantities = {}
    for row, accepted, rejected, short in rows:
        if accepted + rejected + short > row.qty - row.received_qty or row.warehouse != c["source_warehouse"]:
            fail()
        if rejected:
            quarantine = frappe.get_doc("Warehouse", text(c.get("rejected_warehouse")))
            if quarantine.company != c["company"] or quarantine.is_group or quarantine.disabled or quarantine.name in (c["source_warehouse"], c["transit_warehouse"]):
                fail()
        quantities[row.name] = (accepted, rejected, short)
    note = make_purchase_receipt(doc.name)
    note.set("items", [row for row in note.items if row.purchase_order_item in quantities])
    for row in note.items:
        row.qty, row.rejected_qty, short = quantities[row.purchase_order_item]
        row.stock_qty = row.qty * row.conversion_factor
        row.received_qty = row.qty + row.rejected_qty
        row.rejected_warehouse = c.get("rejected_warehouse") if row.rejected_qty else None
    note.cf_shortages = canonical({key: value[2] for key, value in quantities.items() if value[2]})
    note.insert().submit()
    # Use ERPNext's receipt-origin stock reservation contract. Only accepted
    # stock is reserved; the native transaction includes PR, SRE and receipt.
    for item in note.items:
        if not item.qty:
            continue
        reserved = frappe.db.sql("select coalesce(sum(reserved_qty),0) from `tabStock Reservation Entry` where from_voucher_type='Purchase Receipt' and from_voucher_detail_no=%s and docstatus=1 for update", (item.name,))[0][0]
        if reserved < item.stock_qty:
            demand = demand_doc(item.sales_order, r, c)
            demand.create_stock_reservation_entries(from_voucher_type="Purchase Receipt", items_details=[{
                "sales_order_item": item.sales_order_item, "warehouse": item.warehouse,
                "qty_to_reserve": item.stock_qty - reserved, "from_voucher_no": note.name,
                "from_voucher_detail_no": item.name}], notify=False)
    return note.name, {}, None


def reserve(p, r, c):
    shape(p, ("demand", "lines"))
    doc = demand_doc(p["demand"], r, c)
    rows = selected(doc, lines(p, ("demand_line", "qty")), "demand_line")
    details = []
    for row, qty in rows:
        if qty > allocation_budget(row)["reserve_budget"] or row.warehouse != c["source_warehouse"]:
            fail()
        details.append({"sales_order_item": row.name, "warehouse": row.warehouse, "qty_to_reserve": qty * row.conversion_factor})
    doc.create_stock_reservation_entries(items_details=details, notify=False)
    return doc.name, {}, None


def pack(p, r, c):
    shape(p, ("demand", "package_id", "lines"))
    doc = demand_doc(p["demand"], r, c)
    quantities = {}
    for row, qty in selected(doc, lines(p, ("demand_line", "qty")), "demand_line"):
        reserved = frappe.db.sql("select coalesce(sum(reserved_qty-delivered_qty),0) from `tabStock Reservation Entry` where voucher_detail_no=%s and docstatus=1 for update", (row.name,))[0][0]
        if qty > row.qty - row.delivered_qty or qty * row.conversion_factor > reserved or row.warehouse != c["source_warehouse"]:
            fail()
        quantities[row.name] = qty
    note = make_delivery_note(doc.name)
    note.set("items", [row for row in note.items if row.so_detail in quantities])
    note.cf_package = text(p["package_id"])
    if doc.cf_movement == "owned_transfer":
        target = frappe.get_doc("Warehouse", c["transit_warehouse"])
        if target.company != c["company"] or target.is_group or target.disabled or target.name == c["source_warehouse"]:
            fail()
        note.set_target_warehouse = target.name
    for row in note.items:
        row.qty = quantities[row.so_detail]
        row.stock_qty = row.qty * row.conversion_factor
        if doc.cf_movement == "owned_transfer":
            row.target_warehouse = note.set_target_warehouse
    note.insert()
    if doc.cf_movement == "owned_transfer" and any(row.target_warehouse != c["transit_warehouse"] for row in note.items):
        fail()
    return note.name, {}, ["packing", doc.name, note.cf_package]


def dispatch(p, r, c):
    shape(p, ("demand", "delivery"), ("platform_operation",))
    demand = demand_doc(p["demand"], r, c)
    note = lock_doc("Delivery Note", p["delivery"], c, draft=True)
    if note.docstatus != 0 or not note.items or any(row.against_sales_order != demand.name for row in note.items):
        fail()
    if r["source_kind"] == "platform_supply":
        platform = frappe.get_doc(OP, text(p.get("platform_operation")), for_update=True)
        frozen = json.loads(platform.input_json)
        if platform.organization != r["organization"] or platform.binding != r["binding"] or platform.target != demand.name or platform.purpose != "prepare" or platform.stage != "SUCCEEDED" or frozen.get("purpose") != "supply_shipping" or frozen.get("delivery") != note.name:
            fail()
    original = {row.name: row for row in demand.items}
    for row in note.items:
        source = original.get(row.so_detail)
        if not source or row.item_code != source.item_code or row.uom != source.uom or row.conversion_factor != source.conversion_factor or row.warehouse != c["source_warehouse"] or row.qty > source.qty - source.delivered_qty:
            fail()
        quantity(row.qty)
        reserved = frappe.db.sql("select coalesce(sum(reserved_qty-delivered_qty),0) from `tabStock Reservation Entry` where voucher_detail_no=%s and docstatus=1 for update", (source.name,))[0][0]
        if row.qty * row.conversion_factor > reserved:
            fail()
        if demand.cf_movement == "owned_transfer" and row.target_warehouse != c["transit_warehouse"]:
            fail()
    note.submit()
    return note.name, {}, ["physical_package", demand.name, note.cf_package]


def prepare(p, r, c):
    shape(p, ("purpose", "demand", "version", "data"), ("delivery", "selection"))
    demand = demand_doc(p["demand"], r, c)
    if p["purpose"] not in ("accept_address", "read_address", "consumer_tracking", "supply_shipping"):
        fail()
    text(p["version"])
    if not isinstance(p["data"], dict):
        fail()
    if p["purpose"] in ("accept_address", "read_address"):
        shape(p["data"], ("orderNo",), ("handleType",))
        if p["data"]["orderNo"] != r["source_id"]:
            fail()
        if not c.get("address_retention_seconds") or not frappe.conf.get("encryption_key"):
            fail()
    effect = None if p["purpose"] == "read_address" else [p["purpose"], demand.name]
    if p["purpose"] in ("consumer_tracking", "supply_shipping"):
        delivery = lock_doc("Delivery Note", p.get("delivery"), c, draft=True)
        if not delivery.items or any(row.against_sales_order != demand.name for row in delivery.items):
            fail()
        if p["purpose"] == "consumer_tracking":
            effect.append(delivery.name)
    return demand.name, {}, effect


def current_operation(p, r):
    frappe.db.sql("select name from `tabCommerce Fulfillment Operation` where name=%s for update", (text(p["operation"]),))
    doc = frappe.get_doc(OP, p["operation"], for_update=True)
    if any(getattr(doc, key) != r[key] for key in ("organization", "binding", "actor", "member")):
        frappe.throw("Operation binding changed", frappe.PermissionError)
    return doc


def transition(command, p, r, c):
    if command == "record_result":
        shape(p, ("operation", "stage", "result"), ("private",))
    else:
        shape(p, ("operation", "version"))
    doc = current_operation(p, r)
    if doc.purpose != "prepare":
        fail()
    demand_doc(doc.target, r, c)
    expected = {"confirm": ("PREPARED", "CONFIRMED"), "claim": ("CONFIRMED", "CLAIMED"), "mark_dispatching": ("CLAIMED", "DISPATCHING")}
    if command == "cancel":
        if doc.stage not in ("PREPARED", "CONFIRMED") or p["version"] != doc.revision:
            fail()
        doc.stage, doc.effect_key = "CANCELLED", None
        for effect in doc.effects:
            effect.effect_key = None
    elif command in expected:
        before, after = expected[command]
        if p["version"] != doc.revision or doc.stage != before:
            fail()
        doc.stage = after
    else:
        if p["stage"] not in ("SUCCEEDED", "DEFINITIVE_NO_EFFECT", "UNKNOWN") or not isinstance(p["result"], dict):
            fail()
        shape(p["result"], (), ("reason", "trace_id", "shipping_id", "package_ids", "accepted_goods_ids"))
        if "private" in p:
            purpose = json.loads(doc.input_json)["purpose"]
            seconds = c.get("address_retention_seconds")
            if purpose not in ("accept_address", "read_address") or p["stage"] != "SUCCEEDED" or not isinstance(p["private"], dict) or not isinstance(seconds, int) or not 1 <= seconds <= 86400:
                fail()
            private = canonical(p["private"])
            if len(private.encode()) > 32768:
                fail()
            doc.private_data = encrypt(private)
            doc.private_expires = add_to_date(now_datetime(), seconds=seconds)
        value = canonical(p["result"])
        if doc.stage == p["stage"] and doc.result_json == value:
            return receipt(doc)
        if doc.stage != "DISPATCHING":
            fail()
        doc.stage, doc.result_json = p["stage"], value
    doc.save()
    return receipt(doc)


def read_command(command, p, r, c):
    if command == "find_demand":
        shape(p, ())
        source = {key: r[key] for key in ("source_kind", "store", "source_id", "organization")}
        target = frappe.db.get_value(OP, {"effect_key": digest([r["organization"], ["demand", source]]), "stage": "SUCCEEDED"}, "target")
        if target:
            demand_doc(target, r, c)
        return {"demand": target}
    if command == "catalog":
        shape(p, ("page",))
        start = (quantity(p["page"]) - 1) * 20
        items = frappe.get_all("Item", filters={"disabled": 0, "is_stock_item": 1}, fields=["name", "item_name", "stock_uom"], order_by="name", start=start, page_length=20)
        for item in items:
            native = frappe.get_doc("Item", item.name)
            units = {item.stock_uom: 1, **{unit.uom: unit.conversion_factor for unit in native.uoms if unit.conversion_factor and unit.conversion_factor > 0}}
            item["uoms"] = [{"uom": unit, "conversion_factor": factor} for unit, factor in list(units.items())[:20]]
        return {"items": items,
            "suppliers": frappe.get_all("Supplier", filters={"disabled": 0}, fields=["name", "supplier_name"], order_by="name", start=start, page_length=20),
            "source_warehouse": c["source_warehouse"], "movement_kind": c.get("supply_movement"),
            "currency": frappe.db.get_value("Company", c["company"], "default_currency")}
    if command == "delivery_details":
        shape(p, ("demand", "delivery"))
        demand = demand_doc(p["demand"], r, c)
        note = lock_doc("Delivery Note", p["delivery"], c, draft=True)
        original = {row.name: row for row in demand.items}
        if any(row.against_sales_order != demand.name or row.so_detail not in original for row in note.items):
            fail()
        return {"name": note.name, "package_id": note.cf_package, "lines": [{"source_line": original[row.so_detail].cf_source_line, "qty": row.qty} for row in note.items]}
    if command in ("read", "read_private"):
        shape(p, ("operation",))
        doc = current_operation(p, r)
        if doc.stage != "DEFINITIVE_NO_EFFECT" or doc.target:
            validate_operation_target(doc, r, c)
        if command == "read_private":
            if not doc.private_data or not doc.private_expires or get_datetime(doc.private_expires) <= now_datetime():
                fail()
            demand_doc(doc.target, r, c)
            expires = get_datetime(doc.private_expires).replace(tzinfo=ZoneInfo(get_system_timezone())).astimezone(timezone.utc).isoformat()
            return {"private": json.loads(decrypt(doc.private_data)), "expires": expires}
        return receipt(doc)
    if command == "list":
        shape(p, ("page",))
        page = quantity(p["page"])
        return {"items": frappe.get_all(OP, filters={"organization": r["organization"], "binding": r["binding"], "purpose": "bind_demand", "stage": "SUCCEEDED"},
            fields=["name as operation", "target", "creation"], order_by="creation desc", start=(page-1)*20, page_length=20)}
    shape(p, ("demand",), ("purchase_page", "delivery_page", "operation_page", "receipt_page"))
    doc = demand_doc(p["demand"], r, c)
    purchase_names = frappe.db.sql("select distinct parent from `tabPurchase Order Item` where sales_order=%s and docstatus<2 order by parent limit 21 offset %s", (doc.name, (quantity(p.get("purchase_page", 1))-1)*20))
    delivery_names = frappe.db.sql("select distinct parent from `tabDelivery Note Item` where against_sales_order=%s and docstatus<2 order by parent limit 21 offset %s", (doc.name, (quantity(p.get("delivery_page", 1))-1)*20))
    purchases, deliveries = [], []
    for (name,) in purchase_names[:20]:
        purchase = purchase_doc(name, r, c, allow_closed=True)
        receipts = []
        receipt_names = frappe.db.sql("select distinct parent from `tabPurchase Receipt Item` where purchase_order=%s and docstatus=1 order by parent desc limit 21 offset %s", (purchase.name, (quantity(p.get("receipt_page", 1))-1)*20))
        for (receipt_name,) in receipt_names[:20]:
            note = lock_doc("Purchase Receipt", receipt_name, c)
            shortages = json.loads(note.cf_shortages or "{}")
            receipts.append({"name": note.name, "date": str(note.posting_date), "lines": [{"purchase_line": row.purchase_order_item,
                "accepted_qty": row.qty, "rejected_qty": row.rejected_qty, "short_qty": shortages.get(row.purchase_order_item, 0)} for row in note.items if row.purchase_order == purchase.name]})
        purchases.append({"name": name, "supplier": purchase.supplier, "external_order": purchase.cf_external_order,
            "receipts": receipts, "has_more_receipts": len(receipt_names)>20,
            "delivery_date": str(purchase.schedule_date), "status": purchase.status, "currency": purchase.currency,
            "lines": [{"line": row.name, "demand_line": row.sales_order_item, "item_code": row.item_code, "uom": row.uom,
                "qty": row.qty, "received_qty": row.received_qty, "rate": row.rate} for row in purchase.items if row.sales_order == doc.name]})
    for (name,) in delivery_names[:20]:
        note = lock_doc("Delivery Note", name, c, draft=True)
        deliveries.append({"name": name, "docstatus": note.docstatus, "package_id": note.cf_package,
            "lines": [{"demand_line": row.so_detail, "qty": row.qty} for row in note.items if row.against_sales_order == doc.name]})
    operation_names = frappe.get_all(OP, filters={"organization": r["organization"], "binding": r["binding"], "target": doc.name, "purpose": "prepare"}, pluck="name", order_by="creation desc", start=(quantity(p.get("operation_page", 1))-1)*20, page_length=21)
    operations = []
    for name in operation_names[:20]:
        operation = frappe.get_doc(OP, name)
        frozen = json.loads(operation.input_json)
        operations.append({**receipt(operation), "purpose": frozen["purpose"], "delivery": frozen.get("delivery"), "selection": frozen.get("selection", {})})
    return {"name": doc.name, "source": json.loads(doc.cf_source), "movement_kind": doc.cf_movement,
        "operations": operations, "has_more_operations": len(operation_names)>20,
        "purchases": purchases, "deliveries": deliveries, "has_more_purchases": len(purchase_names)>20, "has_more_deliveries": len(delivery_names)>20,
        "lines": [{"line": row.name, "source_line": row.cf_source_line, "item_code": row.item_code, "uom": row.uom,
            "qty": row.qty, "delivered_qty": row.delivered_qty, **allocation_budget(row)} for row in doc.items]}


def validate_operation_target(operation, request, cfg):
    if operation.purpose in ("bind_demand", "reserve", "prepare"):
        demand_doc(operation.target, request, cfg)
    elif operation.purpose in ("create_purchase", "record_external_order", "close_purchase"):
        purchase_doc(operation.target, request, cfg, allow_closed=True)
    elif operation.purpose == "receive":
        doc = lock_doc("Purchase Receipt", operation.target, cfg)
        for row in doc.items:
            purchase_doc(row.purchase_order, request, cfg, allow_closed=True)
    elif operation.purpose in ("pack", "dispatch"):
        doc = lock_doc("Delivery Note", operation.target, cfg, draft=True)
        for row in doc.items:
            demand_doc(row.against_sales_order, request, cfg)
    else:
        fail()


HANDLERS = {"bind_demand": bind_demand, "create_purchase": create_purchase, "record_external_order": record_external_order,
    "receive": receive, "close_purchase": close_purchase, "reserve": reserve, "pack": pack, "dispatch": dispatch, "prepare": prepare}


def purge_private_data():
    # Reuse the native ERP scheduler. Expired data is never returned, even before purge.
    frappe.db.sql("update `tabCommerce Fulfillment Operation` set private_data=null where private_expires<=%s and private_data is not null", (now_datetime(),))
