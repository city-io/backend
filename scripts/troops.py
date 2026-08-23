#!/usr/bin/env python3
"""Dev helper to exercise the ArmyService over the Connect JSON API.

The server speaks Connect, so unary RPCs are plain HTTP POSTs with a JSON body
and Content-Type: application/json. This avoids needing grpcurl.

Usage (server must be running on :8080; see `make all`):

    python3 scripts/troops.py login                 # cache a JWT for the test user
    python3 scripts/troops.py cities                # list owned cities
    python3 scripts/troops.py barracks              # build a barracks in the first owned city
    python3 scripts/troops.py train <barracksId> soldier 5
    python3 scripts/troops.py queue <barracksId>
    python3 scripts/troops.py armies                # list your armies
    python3 scripts/troops.py move <armyId> <x> <y>
    python3 scripts/troops.py getarmy <armyId>
    python3 scripts/troops.py merge <targetId> <sourceId>
    python3 scripts/troops.py city <cityId>
    python3 scripts/troops.py smoke                 # full guided flow (train -> spawn -> move)

Troop types: soldier | archer | cavalry | artillery
"""
import json
import os
import sys
import time
import urllib.request
import urllib.error

BASE = os.environ.get("CITYIO_BASE", "http://localhost:8080")
TOKEN_FILE = "/tmp/cityio_token"
EMAIL = os.environ.get("CITYIO_EMAIL", "cityio@example.com")
PASSWORD = os.environ.get("CITYIO_PASSWORD", "cityio")

TROOP_ENUM = {
    "soldier": "TROOP_TYPE_SOLDIER",
    "archer": "TROOP_TYPE_ARCHER",
    "cavalry": "TROOP_TYPE_CAVALRY",
    "artillery": "TROOP_TYPE_ARTILLERY",
}


def call(proc, body, token=None):
    req = urllib.request.Request(
        BASE + "/" + proc,
        data=json.dumps(body).encode(),
        headers={"Content-Type": "application/json"},
    )
    if token:
        req.add_header("Authorization", "Bearer " + token)
    try:
        with urllib.request.urlopen(req) as r:
            return r.status, json.loads(r.read().decode() or "{}")
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read().decode() or "{}")


def token():
    if not os.path.exists(TOKEN_FILE):
        sys.exit("no token cached; run: python3 scripts/troops.py login")
    return open(TOKEN_FILE).read().strip()


def pp(status, resp):
    print(status, json.dumps(resp, indent=2))


def cmd_login():
    st, r = call("cityio.service.v1.UserService/Login",
                 {"identifier": EMAIL, "password": PASSWORD})
    if st != 200:
        pp(st, r)
        sys.exit(1)
    open(TOKEN_FILE, "w").write(r["token"])
    print("logged in as", EMAIL, "-> token cached at", TOKEN_FILE)


def cmd_cities():
    st, r = call("cityio.service.v1.CityService/ListCities", {}, token())
    for c in r.get("entities", {}).get("cities", []):
        print(c["cityId"]["value"], "start", c["start"], "pop", c.get("population"),
              "milPop", c.get("militaryPopulation"), "foodUpkeep", c.get("foodUpkeep"))


def first_city():
    _, r = call("cityio.service.v1.CityService/ListCities", {}, token())
    return r["entities"]["cities"][0]


def cmd_barracks():
    c = first_city()
    cid = c["cityId"]["value"]
    sx, sy = int(c["start"]["x"]), int(c["start"]["y"])
    st, r = call("cityio.service.v1.BuildingService/CreateBuilding",
                 {"cityId": {"value": cid}, "type": "BUILDING_TYPE_BARRACKS",
                  "coords": {"x": sx, "y": sy}}, token())
    if st != 200:
        pp(st, r)
        return
    print("barracks:", r["building"]["buildingId"]["value"], "at", sx, sy,
          "(waits ~10s to finish construction before it can train)")


def cmd_train(bid, ttype, count):
    st, r = call("cityio.service.v1.ArmyService/TrainTroops",
                 {"barracksId": {"value": bid}, "type": TROOP_ENUM[ttype],
                  "count": int(count)}, token())
    print("train:", st, r)
    return st, r


def cmd_queue(bid):
    st, r = call("cityio.service.v1.ArmyService/ListTrainingOrders",
                 {"barracksId": {"value": bid}}, token())
    pp(st, r)


def cmd_armies():
    st, r = call("cityio.service.v1.ArmyService/ListArmies", {}, token())
    entities = r.get("entities", {})
    armies = entities.get("armies", [])
    orders = {o.get("armyOrderId", {}).get("value"): o
              for o in entities.get("armyOrders", [])}
    print(len(armies), "armies:")
    for a in armies:
        order = orders.get(a.get("orderId", {}).get("value"), {})
        objective = next((order.get(name) for name in ("move", "attackArmy", "conquerSettlement", "retreat") if order.get(name)), {})
        print("  ", a["armyId"]["value"], "at", a["coords"], "dest",
              objective.get("destination") or objective.get("lastKnownCoords"), "troops", a.get("troops"))


def cmd_move(aid, x, y):
    st, r = call("cityio.service.v1.ArmyService/MoveArmy",
                 {"armyId": {"value": aid}, "destination": {"x": int(x), "y": int(y)}}, token())
    print("move:", st, r if st != 200 else "OK")


def cmd_getarmy(aid):
    st, r = call("cityio.service.v1.ArmyService/GetArmy", {"armyId": {"value": aid}}, token())
    pp(st, r)


def cmd_merge(tgt, src):
    st, r = call("cityio.service.v1.ArmyService/MergeArmies",
                 {"targetArmyId": {"value": tgt}, "sourceArmyId": {"value": src}}, token())
    print("merge:", st, r if st != 200 else "OK")


def cmd_city(cid):
    st, r = call("cityio.service.v1.CityService/GetCity", {"cityId": {"value": cid}}, token())
    pp(st, r)


def cmd_smoke():
    cmd_login()
    c = first_city()
    cid = c["cityId"]["value"]
    sx, sy = int(c["start"]["x"]), int(c["start"]["y"])
    print("city", cid, "start", sx, sy)

    st, r = call("cityio.service.v1.BuildingService/CreateBuilding",
                 {"cityId": {"value": cid}, "type": "BUILDING_TYPE_BARRACKS",
                  "coords": {"x": sx, "y": sy}}, token())
    if st != 200:
        pp(st, r); return
    bid = r["building"]["buildingId"]["value"]
    print("barracks", bid, "- waiting ~13s for construction")
    time.sleep(13)

    st, training = cmd_train(bid, "soldier", 5)
    if st != 200:
        return
    aid = training["order"]["armyId"]["value"]
    cmd_queue(bid)
    _, r = call("cityio.service.v1.CityService/GetCity", {"cityId": {"value": cid}}, token())
    print("city milPop after train:", r["city"].get("militaryPopulation"))
    print("waiting ~28s for training (5 soldiers at 5s each)")
    time.sleep(28)

    cmd_armies()
    cmd_move(aid, sx + 4, sy + 4)
    time.sleep(4)
    st, r = call("cityio.service.v1.ArmyService/GetArmy", {"armyId": {"value": aid}}, token())
    entities = r.get("entities", {})
    if entities.get("armies"):
        army = entities["armies"][0]
        order = (entities.get("armyOrders") or [{}])[0]
        objective = order.get("move", {})
        print("army after ~4 move ticks:", army.get("coords"), "dest", objective.get("destination"))
    else:
        print("getarmy:", st, r)


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        return
    cmd, args = sys.argv[1], sys.argv[2:]
    handlers = {
        "login": cmd_login, "cities": cmd_cities, "barracks": cmd_barracks,
        "train": cmd_train, "queue": cmd_queue, "armies": cmd_armies, "move": cmd_move,
        "getarmy": cmd_getarmy, "merge": cmd_merge, "city": cmd_city, "smoke": cmd_smoke,
    }
    if cmd not in handlers:
        print(__doc__); sys.exit(1)
    handlers[cmd](*args)


if __name__ == "__main__":
    main()
