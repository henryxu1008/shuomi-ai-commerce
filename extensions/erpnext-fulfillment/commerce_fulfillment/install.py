import frappe
from frappe.custom.doctype.custom_field.custom_field import create_custom_fields
from frappe.permissions import add_permission, update_permission_property

ROLE = "Commerce Fulfillment Service"


def after_install():
    if not frappe.db.exists("Role", ROLE):
        frappe.get_doc({"doctype": "Role", "role_name": ROLE, "desk_access": 0}).insert()
    fields = {
        "Sales Order": [
            {"fieldname": "cf_source", "label": "Commerce source", "fieldtype": "Small Text", "read_only": 1},
            {"fieldname": "cf_binding", "label": "Commerce binding", "fieldtype": "Data", "read_only": 1},
            {"fieldname": "cf_movement", "label": "Commerce movement", "fieldtype": "Data", "read_only": 1},
        ],
        "Sales Order Item": [{"fieldname": "cf_source_line", "label": "Original platform line", "fieldtype": "Data", "read_only": 1}],
        "Purchase Order": [{"fieldname": "cf_external_order", "label": "Supplier external order", "fieldtype": "Data", "read_only": 1, "allow_on_submit": 1}],
        "Delivery Note": [{"fieldname": "cf_package", "label": "Physical package", "fieldtype": "Data", "read_only": 1}],
    }
    create_custom_fields(fields, update=True)
    for doctype in ["Commerce Fulfillment Operation", "Sales Order", "Purchase Order", "Purchase Receipt", "Delivery Note", "Stock Reservation Entry"]:
        add_permission(doctype, ROLE, permlevel=0)
        for permission in ["read", "write", "create"]:
            update_permission_property(doctype, ROLE, 0, permission, 1)
        if frappe.get_meta(doctype).is_submittable:
            update_permission_property(doctype, ROLE, 0, "submit", 1)
    for doctype in ["Item", "Customer", "Supplier", "Warehouse", "Company", "UOM", "Price List", "Currency", "Bin", "Account"]:
        add_permission(doctype, ROLE, permlevel=0)
        update_permission_property(doctype, ROLE, 0, "read", 1)
