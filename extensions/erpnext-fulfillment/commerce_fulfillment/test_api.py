"""Native integration tests. Run only on a dedicated synthetic ERPNext site."""
import json
import uuid
from unittest.mock import patch

import frappe
from frappe.tests.utils import FrappeTestCase
from frappe.utils import add_days, nowdate

from commerce_fulfillment.api import execute


class TestFulfillment(FrappeTestCase):
    def setUp(self):
        if not frappe.conf.get("commerce_fulfillment_test_site"):
            self.skipTest("requires explicitly designated isolated test site")
        self.suffix = uuid.uuid4().hex[:8]
        self.company = frappe.get_doc({"doctype": "Company", "company_name": "CF " + self.suffix,
            "abbr": self.suffix, "default_currency": "USD", "country": "United States",
            "enable_perpetual_inventory": 0}).insert()
        self.warehouse = frappe.db.get_value("Warehouse", {"company": self.company.name, "warehouse_name": "Stores"}, "name")
        self.transit = frappe.get_doc({"doctype": "Warehouse", "warehouse_name": "CF Transit " + self.suffix,
            "company": self.company.name, "warehouse_type": "Transit"}).insert().name
        customer_group = frappe.db.get_value("Customer Group", {"is_group": 0}, "name")
        territory = frappe.db.get_value("Territory", {"is_group": 0}, "name")
        def customer(internal):
            return frappe.get_doc({"doctype": "Customer", "customer_name": "CF Customer " + str(internal) + self.suffix,
                "customer_type": "Company", "customer_group": customer_group, "territory": territory,
                "is_internal_customer": internal, "represents_company": self.company.name if internal else None,
                "companies": [{"company": self.company.name}] if internal else []}).insert().name
        self.customer, self.internal = customer(0), customer(1)
        self.supplier = frappe.get_doc({"doctype": "Supplier", "supplier_name": "CF Supplier " + self.suffix,
            "supplier_group": frappe.db.get_value("Supplier Group", {"is_group": 0}, "name"), "supplier_type": "Company"}).insert().name
        self.item = frappe.get_doc({"doctype": "Item", "item_code": "CF-" + self.suffix,
            "item_name": "Synthetic goods", "item_group": "Products", "stock_uom": "Nos", "is_stock_item": 1}).insert().name
        self.previous = frappe.conf.get("commerce_fulfillment")
        frappe.conf.commerce_fulfillment = {"organization": "test-org", "company": self.company.name,
            "customer": self.customer, "internal_customer": self.internal, "source_warehouse": self.warehouse,
            "transit_warehouse": self.transit, "service_user": "Administrator", "test_only": True,
            "supply_movement": "owned_transfer", "address_retention_seconds": 3600,
            "selling_price_list": frappe.db.get_value("Price List", {"selling": 1, "currency": "USD"}, "name"),
            "buying_price_list": frappe.db.get_value("Price List", {"buying": 1, "currency": "USD"}, "name")}

    def tearDown(self):
        frappe.set_user("Administrator")
        frappe.conf.commerce_fulfillment = self.previous
        super().tearDown()

    def command(self, command, payload, key=None):
        return execute({"operation_id": key or str(uuid.uuid4()), "organization": "test-org",
            "actor": "test-user", "member": "test-member", "binding": "a" * 64,
            "store": "test-store", "source_kind": getattr(self, "source_kind", "consumer_order"), "source_id": "source-" + self.suffix,
            "command": command, "payload": payload})

    def demand(self, kind="consumer_order", quantity=5):
        self.source_kind = kind
        return {"source_kind": kind, "store": "test-store", "source_id": "source-" + self.suffix,
            "movement_kind": "owned_transfer" if kind == "platform_supply" else "commercial_sale",
            "delivery_date": add_days(nowdate(), 5), "lines": [{"source_line": "source-line", "item_code": self.item,
                "qty": quantity, "uom": "Nos", "warehouse": self.warehouse}]}

    def test_demand_replay_and_conflict(self):
        key = str(uuid.uuid4())
        payload = self.demand()
        first = self.command("bind_demand", payload, key)
        self.assertEqual(first, self.command("bind_demand", payload, key))
        payload["lines"][0]["qty"] = 6
        with self.assertRaises(frappe.ValidationError):
            self.command("bind_demand", payload, key)
        self.assertEqual(frappe.get_doc("Sales Order", first["target"]).items[0].qty, 5)

    def test_procurement_partial_receipt_and_owned_dispatch(self):
        demand = self.command("bind_demand", self.demand("platform_supply"))
        line = frappe.get_doc("Sales Order", demand["target"]).items[0].name
        po = self.command("create_purchase", {"demand": demand["target"], "supplier": self.supplier,
            "delivery_date": add_days(nowdate(), 5), "lines": [{"demand_line": line, "qty": 5, "rate": 4}]})
        purchase = frappe.get_doc("Purchase Order", po["target"])
        self.command("record_external_order", {"purchase": purchase.name, "external_order": "manual-supplier-1"})
        self.command("receive", {"purchase": purchase.name, "lines": [{"purchase_line": purchase.items[0].name, "qty": 3}]})
        self.assertEqual(frappe.get_doc("Purchase Order", purchase.name).items[0].received_qty, 3)
        with self.assertRaises(frappe.ValidationError):
            self.command("receive", {"purchase": purchase.name, "lines": [{"purchase_line": purchase.items[0].name, "qty": 3}]})
        self.command("reserve", {"demand": demand["target"], "lines": [{"demand_line": line, "qty": 3}]})
        packed = self.command("pack", {"demand": demand["target"], "package_id": "physical-package-1",
            "lines": [{"demand_line": line, "qty": 2}]})
        self.assertEqual(frappe.get_doc("Delivery Note", packed["target"]).docstatus, 0)
        self.assertEqual(frappe.db.get_value("Bin", {"item_code": self.item, "warehouse": self.warehouse}, "actual_qty"), 3)
        result = self.command("dispatch", {"demand": demand["target"], "delivery": packed["target"]})
        note = frappe.get_doc("Delivery Note", result["target"])
        self.assertEqual(note.items[0].target_warehouse, self.transit)
        self.assertEqual(frappe.get_doc("Sales Order", demand["target"]).items[0].delivered_qty, 2)
        for warehouse, qty in [(self.warehouse, 1), (self.transit, 2)]:
            self.assertEqual(frappe.db.get_value("Bin", {"item_code": self.item, "warehouse": warehouse}, "actual_qty"), qty)
        detail = self.command("details", {"demand": demand["target"]})
        self.assertEqual(detail["purchases"][0]["lines"][0]["received_qty"], 3)
        self.assertEqual(detail["deliveries"][0]["docstatus"], 1)

    def test_unknown_effect_cannot_be_bypassed_with_new_operation(self):
        demand = self.command("bind_demand", self.demand())
        payload = {"purpose": "accept_address", "demand": demand["target"], "version": "v1", "data": {"orderNo": "source-" + self.suffix}}
        operation = str(uuid.uuid4())
        self.command("prepare", payload, operation)
        for command in ["confirm", "claim", "mark_dispatching"]:
            self.command(command, {"operation": operation, "version": "v1"})
        self.command("record_result", {"operation": operation, "stage": "UNKNOWN", "result": {"reason": "response_lost"}})
        with self.assertRaises(frappe.ValidationError):
            self.command("prepare", payload)
        result = self.command("read", {"operation": operation})
        self.assertEqual(result["stage"], "UNKNOWN")

    def test_failure_rolls_back_native_document_and_receipt(self):
        key = str(uuid.uuid4())
        from commerce_fulfillment import api
        original = api.HANDLERS["bind_demand"]

        def fault(*args):
            original(*args)
            raise RuntimeError("synthetic lost response before transaction commit")

        with patch.dict(api.HANDLERS, {"bind_demand": fault}):
            with self.assertRaises(RuntimeError):
                self.command("bind_demand", self.demand(), key)
        self.assertFalse(frappe.db.exists("Commerce Fulfillment Operation", key))
        self.assertEqual(frappe.db.count("Sales Order", {"company": self.company.name}), 0)
        result = self.command("bind_demand", self.demand(), key)
        self.assertEqual(result["stage"], "SUCCEEDED")

    def test_wrong_org_and_actor_cannot_reuse_receipt(self):
        key = str(uuid.uuid4())
        self.command("bind_demand", self.demand(), key)
        request = {"operation_id": key, "organization": "other-org", "actor": "test-user", "member": "test-member",
            "store": "test-store", "source_kind": "consumer_order", "source_id": "source-" + self.suffix,
            "binding": "a" * 64, "command": "bind_demand", "payload": self.demand()}
        with self.assertRaises(frappe.PermissionError):
            execute(request)
        request["organization"], request["actor"] = "test-org", "other-actor"
        with self.assertRaises(frappe.ValidationError):
            execute(request)

    def test_service_role_can_run_native_path_without_financial_reads(self):
        from commerce_fulfillment.install import ROLE
        user = frappe.get_doc({"doctype": "User", "email": "fulfillment-" + self.suffix + "@example.invalid",
            "first_name": "Synthetic service", "send_welcome_email": 0, "roles": [{"role": ROLE}]}).insert()
        frappe.conf.commerce_fulfillment["service_user"] = user.name
        frappe.set_user(user.name)
        result = self.command("bind_demand", self.demand())
        line = frappe.get_doc("Sales Order", result["target"]).items[0].name
        po = self.command("create_purchase", {"demand": result["target"], "supplier": self.supplier,
            "delivery_date": add_days(nowdate(), 5), "lines": [{"demand_line": line, "qty": 5, "rate": 4}]})
        self.assertTrue(po["target"])
        self.assertFalse(frappe.has_permission("Bank Account", "read"))

    def test_addresses_are_encrypted_and_version_checked(self):
        demand = self.command("bind_demand", self.demand())
        operation = str(uuid.uuid4())
        self.command("prepare", {"purpose": "accept_address", "demand": demand["target"],
            "version": "v1", "data": {"orderNo": "source-" + self.suffix}}, operation)
        with self.assertRaises(frappe.ValidationError):
            self.command("confirm", {"operation": operation, "version": "stale-version"})
        for command in ["confirm", "claim", "mark_dispatching"]:
            self.command(command, {"operation": operation, "version": "v1"})
        address = {"recipient": "Synthetic private name", "address": "Synthetic private address"}
        result = self.command("record_result", {"operation": operation, "stage": "SUCCEEDED", "result": {}, "private": address})
        doc = frappe.get_doc("Commerce Fulfillment Operation", operation)
        self.assertNotIn("Synthetic private", doc.as_json())
        self.assertNotIn("Synthetic private", json.dumps(result))
        self.assertEqual(self.command("read_private", {"operation": operation})["private"], address)

    def test_current_source_cannot_be_substituted_for_another_demand(self):
        demand = self.command("bind_demand", self.demand())
        request = {"operation_id": str(uuid.uuid4()), "organization": "test-org", "actor": "test-user", "member": "test-member",
            "binding": "a" * 64, "store": "test-store", "source_kind": "consumer_order", "source_id": "different-source",
            "command": "details", "payload": {"demand": demand["target"]}}
        with self.assertRaises(frappe.ValidationError):
            execute(request)

    def test_unknown_goods_effect_survives_new_package_and_operation_ids(self):
        demand = self.command("bind_demand", self.demand(quantity=1))
        line = frappe.get_doc("Sales Order", demand["target"]).items[0].name
        frappe.get_doc({"doctype": "Stock Entry", "stock_entry_type": "Material Receipt", "company": self.company.name,
            "items": [{"item_code": self.item, "t_warehouse": self.warehouse, "qty": 1, "basic_rate": 4, "conversion_factor": 1}]}).insert().submit()
        self.command("reserve", {"demand": demand["target"], "lines": [{"demand_line": line, "qty": 1}]})
        def packing(package):
            return self.command("pack", {"demand": demand["target"], "package_id": package, "lines": [{"demand_line": line, "qty": 1}]})["target"]
        first = packing("first-package")
        operation = str(uuid.uuid4())
        self.command("prepare", {"purpose": "consumer_tracking", "demand": demand["target"], "delivery": first, "version": "v1", "data": {}}, operation)
        for command in ["confirm", "claim", "mark_dispatching"]:
            self.command(command, {"operation": operation, "version": "v1"})
        self.command("record_result", {"operation": operation, "stage": "UNKNOWN", "result": {"reason": "lost_response"}})
        second = packing("different-package")
        with self.assertRaises(frappe.ValidationError):
            self.command("prepare", {"purpose": "consumer_tracking", "demand": demand["target"], "delivery": second, "version": "v2", "data": {}})

    def test_purchase_receipt_can_be_recovered_by_original_operation_key(self):
        demand = self.command("bind_demand", self.demand())
        line = frappe.get_doc("Sales Order", demand["target"]).items[0].name
        key = str(uuid.uuid4())
        purchase = self.command("create_purchase", {"demand": demand["target"], "supplier": self.supplier,
            "delivery_date": add_days(nowdate(), 5), "lines": [{"demand_line": line, "qty": 5, "rate": 4}]}, key)
        self.assertEqual(self.command("read", {"operation": key}), purchase)
