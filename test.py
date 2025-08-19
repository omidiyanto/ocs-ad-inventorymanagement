import requests
import urllib3
import json

# Nonaktifkan peringatan SSL (opsional)
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

def login_to_admanager():
    """
    Fungsi untuk login ke ManageEngine AD Manager Plus.
    Mengembalikan objek session jika berhasil, None jika gagal.
    """
    print("--- MEMULAI PROSES LOGIN ---")
    base_url = "http://192.168.88.73:8080"
    username = "admin"
    encrypted_password = "LI1KrLGD9RgrP3BSD05o9V82TrQF8U8tly+Abv2420as0R5d55dm1PTh6cJ4FdiPqWFRFesKlyVayd885vV+DtAq4X6CPm5qB1SaumipPlL6zfrJF8eqFurBElnj2p8fKwMZ4sjGpyKA8VmUJgHsaOgc3tB7ECRtfsCFFjnZ5vg="

    session = requests.Session()
    session.headers.update({
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36'
    })

    print(f"[*] Langkah 1: Mengunjungi {base_url}/ untuk mendapatkan cookie...")
    try:
        init_response = session.get(base_url + "/", verify=False)
        if init_response.status_code == 200 and 'admpcsrf' in session.cookies:
            print("[+] Cookie awal berhasil didapatkan.")
        else:
            print("[-] Gagal mendapatkan cookie awal.")
            return None
    except requests.exceptions.RequestException as e:
        print(f"[!] Terjadi kesalahan koneksi pada Langkah 1: {e}")
        return None

    print(f"\n[*] Langkah 2: Mencoba login...")
    login_url = base_url + "/j_security_check"
    params = {'LogoutFromSSO': 'true'}
    payload = {
        'is_admp_pass_encrypted': 'true',
        'j_username': username,
        'j_password': encrypted_password,
        'domainName': 'ADManager Plus Authentication',
        'AUTHRULE_NAME': 'ADAuthenticator'
    }

    try:
        login_response = session.post(login_url, params=params, data=payload, verify=False)
        if login_response.status_code == 200 and ("Dashboard" in login_response.text or "Log Out" in login_response.text or "index.do" in login_response.url):
            print("[SUCCESS] Login berhasil! Sesi valid.")
            print("---------------------------\n")
            return session
        else:
            print("[FAILED] Login gagal.")
            return None
    except requests.exceptions.RequestException as e:
        print(f"[!] Terjadi kesalahan koneksi pada Langkah 2: {e}")
        return None

def transform_report_data(raw_data: dict):
    """
    Mengubah data JSON mentah dari ADManager menjadi format yang lebih sederhana.
    """
    print("[*] Memulai transformasi data JSON...")
    simplified_list = []
    
    # Kunci 'resultrows' (huruf kecil) berisi data yang kita inginkan
    if 'resultrows' not in raw_data:
        print("[!] Kunci 'resultrows' tidak ditemukan dalam data mentah.")
        return simplified_list

    # Membuat pemetaan dari ATTRIB_ID ke nama field yang kita inginkan
    # Berdasarkan contoh output: 3001=Nama, 3019=Waktu Logon, 3021=Status
    id_to_key_map = {
        3001: 'computer_name',
        3019: 'last_logon_time',
        3021: 'computer_status'
    }

    for row in raw_data['resultrows']:
        new_item = {}
        # Setiap baris memiliki list 'COLUMNS', kita proses satu per satu
        for column in row['COLUMNS']:
            attrib_id = column.get('ATTRIB_ID')
            if attrib_id in id_to_key_map:
                key = id_to_key_map[attrib_id]
                value = column.get('VALUE')
                
                # Khusus untuk status, kita ubah ke huruf kecil
                if key == 'computer_status':
                    new_item[key] = value.lower()
                else:
                    new_item[key] = value
        
        # Tambahkan dictionary yang sudah jadi ke dalam list utama
        if new_item:
            simplified_list.append(new_item)
    
    print("[+] Transformasi data selesai.")
    return simplified_list

def fetch_report(session: requests.Session):
    """
    Mengambil data laporan dan memprosesnya menjadi format yang sederhana.
    """
    print("--- MEMULAI PENGAMBILAN LAPORAN ---")
    base_url = "http://192.168.88.73:8080"
    report_url = base_url + "/api/json/reports/report/getReportResultRows"
    
    try:
        csrf_token = session.cookies['admpcsrf']
    except KeyError:
        print("[!] Tidak dapat menemukan 'admpcsrf' cookie di session.")
        return

    params_dict = {
        "pageNavigateData": {"startIndex": 1, "toIndex": 2, "rangeList": [25, 50, 75, 100], "range": 2, "totalCount": 0, "isNavigate": False},
        "searchText": {}, "searchCriteriaType": {}, "sortAttribId": -1, "sortingOrder": True, "reportResultFilter": {}, "rvcFilter": {}, "viewOf": "default",
        "dbFilterDetails": {"objectId": 3, "filters": []}
    }

    report_payload = {
        'reportId': '210', 'generationId': '1021', 'params': json.dumps(params_dict),
        'intersect': 'false', 'admpcsrf': csrf_token
    }

    print(f"[*] Mengirim permintaan POST ke: {report_url}")
    try:
        response = session.post(report_url, data=report_payload, verify=False)
        
        if response.status_code == 200:
            print("[SUCCESS] Berhasil mengambil data laporan!")
            try:
                raw_report_data = response.json()
                
                # Panggil fungsi transformasi di sini
                clean_data = transform_report_data(raw_report_data)

                print("\n--- HASIL DATA YANG SUDAH DIUBAH ---")
                print(json.dumps(clean_data, indent=4))

            except json.JSONDecodeError:
                print("[-] Respon berhasil (200 OK), tetapi bukan format JSON yang valid.")
        else:
            print(f"[FAILED] Gagal mengambil laporan. Status Code: {response.status_code}")
            print("Response Text:", response.text)

    except requests.exceptions.RequestException as e:
        print(f"[!] Terjadi kesalahan koneksi saat mengambil laporan: {e}")

if __name__ == "__main__":
    authenticated_session = login_to_admanager()
    if authenticated_session:
        fetch_report(authenticated_session)
    else:
        print("\nProses dihentikan karena login gagal.")