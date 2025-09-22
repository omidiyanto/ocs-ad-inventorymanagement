from ldap3 import Server, Connection, ALL
from datetime import datetime, timedelta

LDAP_SERVER = "192.168.0.1"
LDAP_PORT = 389
LDAP_USER = "COMPANY\\USERNAME"   
LDAP_PASSWORD = "PASS"
BASE_DN = "DC=example,DC=com"

# Konversi lastLogonTimestamp dari Windows filetime ke Python datetime
def filetime_to_dt(filetime):
    if filetime and filetime.isdigit():
        return datetime(1601, 1, 1) + timedelta(microseconds=int(filetime) // 10)
    return None

# Decode userAccountControl untuk status komputer
def decode_uac(uac_value):
    uac_value = int(uac_value)
    if uac_value & 2:  # bit 2 = ACCOUNTDISABLE
        return "Disabled"
    return "Enabled"

def get_computers():
    server = Server(LDAP_SERVER, port=LDAP_PORT, get_info=ALL)
    conn = Connection(server, user=LDAP_USER, password=LDAP_PASSWORD, auto_bind=True)

    conn.search(
        search_base=BASE_DN,
        search_filter="(objectClass=computer)",
        attributes=["cn", "dNSHostName", "lastLogonTimestamp", "userAccountControl", "whenChanged"]
    )

    for entry in conn.entries:
        computer_name = str(entry.cn)
        last_logon = filetime_to_dt(str(entry.lastLogonTimestamp)) if "lastLogonTimestamp" in entry else None
        status = decode_uac(str(entry.userAccountControl)) if "userAccountControl" in entry else "Unknown"
        last_modified = str(entry.whenChanged) if "whenChanged" in entry else None

        print(f"Computer Name     : {computer_name}")
        print(f"Last Logon Time   : {last_logon}")
        print(f"Computer Status   : {status}")
        print(f"AD Last Modified  : {last_modified}")
        print("-" * 60)

    conn.unbind()

if __name__ == "__main__":
    get_computers()
