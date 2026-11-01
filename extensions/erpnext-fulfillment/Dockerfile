ARG ERPNEXT_IMAGE=frappe/erpnext:v15.124.3@sha256:02c7443608b6af7d088a512aba3a62ecafdba2346be853ddce577194f856af49
FROM ${ERPNEXT_IMAGE}
COPY --chown=1000:1000 . /home/frappe/frappe-bench/apps/commerce_fulfillment
USER frappe
WORKDIR /home/frappe/frappe-bench
RUN env/bin/pip install --no-deps --no-build-isolation -e apps/commerce_fulfillment \
    && env/bin/python -c "from pathlib import Path; p=Path('sites/apps.txt'); apps=p.read_text().splitlines(); p.write_text('\n'.join(dict.fromkeys(apps+['commerce_fulfillment']))+'\n')"
