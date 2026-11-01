app_name = "commerce_fulfillment"
app_title = "Commerce Fulfillment"
app_publisher = "Shuomi Commerce"
app_description = "Bounded native ERPNext fulfillment commands"
required_apps = ["erpnext"]
after_install = "commerce_fulfillment.install.after_install"
scheduler_events = {"cron": {"* * * * *": ["commerce_fulfillment.api.purge_private_data"]}}
