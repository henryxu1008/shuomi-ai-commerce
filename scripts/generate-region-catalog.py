"""Build the bundled region catalog from a pinned set of GeoNames dumps."""
import argparse
from collections import Counter
import gzip
import hashlib
import io
import json
from pathlib import Path
import unicodedata
import zipfile


SOURCES = ("countryInfo.txt", "admin1CodesASCII.txt", "admin2Codes.txt", "cities500.zip", "CN.zip")


def rows(stream):
    for line in stream:
        if line.strip() and not line.startswith("#"):
            yield line.rstrip("\r\n").split("\t")


def text_rows(path):
    with path.open(encoding="utf-8") as stream:
        yield from rows(stream)


def zip_rows(path, name):
    with zipfile.ZipFile(path) as archive, archive.open(name) as data:
        yield from rows(io.TextIOWrapper(data, encoding="utf-8"))


def valid_name(name):
    return bool(name) and len(name.encode()) <= 128 and not any(unicodedata.category(c) == "Cc" for c in name)


def options(records):
    counts = Counter(name for _, name, _ in records)
    labeled = [(ident, f"{name} · {parent}" if counts[name] > 1 and parent else name) for ident, name, parent in records]
    duplicates = Counter(name for _, name in labeled)
    result = []
    for ident, name in labeled:
        if duplicates[name] > 1:
            name = f"{name} · {ident}"
        if not valid_name(name):
            raise ValueError(f"invalid catalog display label: {ident}")
        result.append({"id": ident, "name": name})
    if len(result) != len({item["name"] for item in result}) or len(result) != len({item["id"] for item in result}):
        raise ValueError("duplicate catalog ID or display label")
    return sorted(result, key=lambda item: (item["name"].casefold(), item["id"]))


def build(source):
    zh = {}
    for row in zip_rows(source / "CN.zip", "CN.txt"):
        if row[2] not in ("zh-CN", "zh-Hans", "zh") or not valid_name(row[3].strip()):
            continue
        flags = row[4:8] + [""] * max(0, 8 - len(row))
        if flags[2] == "1" or flags[3] == "1":
            continue
        rank = (flags[0] != "1", ("zh-CN", "zh-Hans", "zh").index(row[2]), row[3])
        if row[1] not in zh or rank < zh[row[1]][0]:
            zh[row[1]] = (rank, row[3].strip())

    def local_name(ident, name):
        return zh.get(ident, (None, name.strip()))[1]

    countries = {r[0]: {"id": r[0], "name": r[4].strip(), "provinces": [], "cities": []}
                 for r in text_rows(source / "countryInfo.txt")}
    subdivisions = {}
    for code, name, _, ident in text_rows(source / "admin1CodesASCII.txt"):
        country = code.split(".")[0]
        if country in countries and country != "SG":
            subdivisions[code] = (ident, local_name(ident, name), [])
    districts = {r[0]: local_name(r[3], r[1]) for r in text_rows(source / "admin2Codes.txt")}
    direct = {country: [] for country in countries}
    for row in zip_rows(source / "cities500.zip", "cities500.txt"):
        ident, name, country, admin1, admin2 = row[0], row[1], row[8], row[10], row[11]
        if country not in countries or row[7] not in {"PPL", "PPLC", "PPLA", "PPLA2", "PPLA3", "PPLA4", "PPLA5"}:
            continue
        if country == "SG" and ident != "1880252":
            # The preference refers to the city-state, not a planning district.
            continue
        subdivision = subdivisions.get(f"{country}.{admin1}")
        bucket = subdivision[2] if subdivision else direct[country]
        bucket.append((ident, local_name(ident, name), districts.get(f"{country}.{admin1}.{admin2}", "")))

    for code, (ident, name, city_rows) in subdivisions.items():
        country = countries[code.split(".")[0]]
        country["provinces"].append({"id": ident, "name": name, "cities": options(city_rows)})
    for code, country in countries.items():
        country["cities"] = options(direct[code])
        provinces = country["provinces"]
        labels = {p["id"]: p["name"] for p in options([(p["id"], p["name"], "") for p in provinces])}
        for province in provinces:
            province["name"] = labels[province["id"]]
        provinces.sort(key=lambda p: (p["name"].casefold(), p["id"]))
    return sorted(countries.values(), key=lambda country: country["id"])


def hash_file(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path, help="directory containing the five GeoNames dump files")
    args = parser.parse_args()
    target = Path(__file__).resolve().parents[1] / "internal" / "geography"
    sources = {name: hash_file(args.source / name) for name in SOURCES}
    countries = build(args.source)
    canonical = json.dumps(countries, ensure_ascii=False, separators=(",", ":")).encode()
    version = "gn-" + hashlib.sha256(canonical).hexdigest()[:16]
    data = json.dumps({"version": version, "countries": countries}, ensure_ascii=False, separators=(",", ":")).encode()
    (target / "catalog.json.gz").write_bytes(gzip.compress(data, mtime=0))
    (target / "sources.json").write_text(json.dumps({"version": version, "sha256": sources}, indent=2) + "\n", encoding="utf-8")
    city_count = lambda c: len(c["cities"]) + sum(len(p["cities"]) for p in c["provinces"])
    print(version, len(countries), sum(map(city_count, countries)), "places; max uncompressed country bytes:", max(len(json.dumps(c, ensure_ascii=False).encode()) for c in countries))


if __name__ == "__main__":
    main()
