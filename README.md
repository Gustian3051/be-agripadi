# AgriPadi AI Backend v2

Backend v2 AgriPadi AI adalah REST API berbasis **Go**, **Gin**, **GORM**, dan **PostgreSQL** untuk mendukung aplikasi mobile smart farming AgriPadi AI. Backend ini menangani autentikasi pengguna, profil, upload gambar, riwayat percakapan, workflow diagnosis hama padi, rule engine sistem pakar, rekomendasi pestisida, dan integrasi LLM.

Backend ini dirancang untuk mode utama:

```text
CNN_MODE=on_device_flutter
```

Artinya proses klasifikasi gambar hama padi dilakukan di aplikasi Flutter menggunakan model **MobileNetV3-Small TensorFlow Lite**, sedangkan backend menerima hasil deteksi dari Flutter dan melanjutkannya ke workflow diagnosis serta rekomendasi.

---

## Ringkasan Teknologi

| Bagian | Teknologi |
|---|---|
| Bahasa | Go |
| HTTP Framework | Gin |
| ORM | GORM |
| Database | PostgreSQL |
| Auth | JWT |
| Config | Viper + config.yml + environment variable |
| LLM | OpenAI-compatible API / Groq |
| Deployment | Docker / Docker Compose |
| Dataset Sistem Pakar | CSV |

---

## Fitur Backend

- Register, login, logout.
- JWT authentication middleware.
- Profil pengguna dan ubah password.
- Upload gambar hama.
- Penyajian static file `/uploads`.
- Manajemen riwayat percakapan.
- Penyimpanan pesan user dan assistant.
- Workflow diagnosis hama padi.
- Sistem pakar berbasis dataset CSV.
- Rekomendasi pestisida berbasis:
  - hama sasaran,
  - gejala,
  - tingkat keparahan,
  - fase pertumbuhan padi,
  - aturan pestisida,
  - waktu aplikasi,
  - skor keamanan.
- Integrasi LLM untuk penjelasan konsultasi dan rekomendasi.
- Docker-ready untuk production dan CI/CD.

---

## Struktur Folder

```text
backend-v2/
├── cmd/
│   └── app/
│       └── main.go
├── config/
│   ├── config.go
│   └── database.go
├── dataset/
│   ├── growth_stage.csv
│   ├── pest.csv
│   ├── pesticide.csv
│   ├── pesticide_application_timing_reference.csv
│   ├── rules.csv
│   ├── severity.csv
│   └── symptom_rule_simple.csv
├── internal/
│   ├── domain/
│   ├── dto/
│   ├── handler/
│   ├── middleware/
│   ├── repository/
│   ├── router/
│   └── service/
│       ├── app/
│       ├── auth/
│       ├── cnn/
│       ├── consultation/
│       ├── expert_system/
│       └── profile/
├── logger/
├── config.yml
├── Dockerfile
├── go.mod
└── go.sum
```

---

## Konfigurasi

Backend menggunakan kombinasi:

1. `config.yml` untuk konfigurasi non-rahasia.
2. Environment variable untuk credential sensitif.
3. `viper.AutomaticEnv()` dan `viper.BindEnv()` untuk membaca environment variable.

Credential penting **jangan ditulis langsung di `config.yml`**.

### Environment Variable

| Variable | Wajib | Keterangan |
|---|---:|---|
| `APP_ENV` | Tidak | Default: `production` |
| `APP_PORT` | Tidak | Port aplikasi. Pada `config.yml` saat ini: `8182` |
| `DATABASE_DSN` | Ya | DSN PostgreSQL |
| `DB_DSN` | Alternatif | Alias untuk `DATABASE_DSN` |
| `JWT_SECRET` | Ya untuk production | Secret JWT |
| `AUTH_JWT_SECRET` | Alternatif | Alias untuk `JWT_SECRET` |
| `LLM_PROVIDER` | Tidak | Default: `groq` |
| `LLM_API_URL` | Tidak | Default: `https://api.groq.com/openai/v1` |
| `LLM_API_KEY` | Disarankan | API key LLM |
| `GROQ_API_KEY` | Alternatif | Alias untuk `LLM_API_KEY` |
| `LLM_MODEL` | Tidak | Default: `qwen/qwen3-32b` |
| `LLM_TIMEOUT` | Tidak | Default: `30s` |
| `CNN_MODE` | Tidak | Default: `on_device_flutter` |
| `CNN_URL` | Tidak | Diisi jika memakai server CNN |
| `LOG_LEVEL` | Tidak | Contoh: `info`, `debug` |
| `LOG_DEBUG` | Tidak | `true` atau `false` |

### Contoh `.env`

Simpan file `.env` di server pada folder yang sama dengan `docker-compose.yml`.

```env
APP_ENV=production
APP_PORT=8182

DATABASE_DSN=postgres://postgres:password_database@postgres:5432/agripadi?sslmode=disable
JWT_SECRET=isi_secret_panjang_minimal_32_karakter

LLM_PROVIDER=groq
LLM_API_URL=https://api.groq.com/openai/v1
LLM_API_KEY=isi_api_key_llm
LLM_MODEL=qwen/qwen3-32b
LLM_TIMEOUT=30s

CNN_MODE=on_device_flutter
CNN_URL=

LOG_LEVEL=info
LOG_DEBUG=false
```

Jika PostgreSQL berjalan sebagai service Docker Compose bernama `postgres`, maka host pada DSN adalah:

```text
postgres
```

Bukan:

```text
localhost
```

Contoh benar:

```env
DATABASE_DSN=postgres://postgres:password_database@postgres:5432/agripadi?sslmode=disable
```

---

## Lokasi `.env` di Server

Jika folder deployment server adalah:

```bash
/root/agripadi
```

Maka struktur yang benar:

```text
/root/agripadi/
├── docker-compose.yml
└── .env
```

Letakkan `.env` di:

```bash
/root/agripadi/.env
```

Backend tidak membaca file `.env` secara langsung dari kode Go. Docker Compose membaca `.env`, lalu memasukkan isinya ke container melalui `env_file`.

---

## Contoh Docker Compose

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: agripadi-postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: agripadi
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password_database
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d agripadi"]
      interval: 10s
      timeout: 5s
      retries: 5

  backend-v2:
    image: ghcr.io/gustian3051/backend-v2:latest
    container_name: agripadi-backend-v2
    restart: unless-stopped
    env_file:
      - .env
    ports:
      - "8282:8182"
    depends_on:
      postgres:
        condition: service_healthy
    volumes:
      - uploads_data:/app/uploads

volumes:
  postgres_data:
  uploads_data:
```

Catatan port:

```text
APP_PORT=8182
ports: "8282:8182"
```

Artinya backend berjalan di port `8182` di dalam container, dan diakses dari luar server melalui port `8282`.

---

## Menjalankan Backend Secara Lokal

Masuk ke folder backend:

```bash
cd backend-v2
```

Set environment variable:

```bash
export APP_ENV=development
export APP_PORT=8182
export DATABASE_DSN="postgres://postgres:password_database@localhost:5432/agripadi?sslmode=disable"
export JWT_SECRET="local_secret_minimal_32_karakter"
export LLM_API_KEY="isi_api_key_llm"
export CNN_MODE="on_device_flutter"
```

Jalankan:

```bash
go run ./cmd/app
```

Cek health check:

```bash
curl http://localhost:8182/health
```

Response normal:

```json
{
  "message": "ok",
  "status": "success"
}
```

---

## Build Docker Image

Dari folder `backend-v2`:

```bash
docker build -t agripadi-backend-v2 .
```

Jalankan container manual:

```bash
docker run --rm \
  --name agripadi-backend-v2 \
  --env-file .env \
  -p 8282:8182 \
  agripadi-backend-v2
```

---

## Deployment CI/CD

Alur production yang disarankan:

```text
GitHub Repository
        ↓
GitHub Actions build Docker image
        ↓
Push image ke GHCR
        ↓
Server pull image terbaru
        ↓
docker compose up -d backend-v2
        ↓
Container membaca env dari /root/agripadi/.env
```

Contoh perintah deploy di server:

```bash
cd /root/agripadi
docker compose pull backend-v2
docker compose up -d backend-v2
docker compose logs -f backend-v2
```

Jangan commit file `.env` ke GitHub.

Tambahkan ke `.gitignore`:

```gitignore
.env
.env.*
!.env.example
uploads/
```

---

## Endpoint API

Base URL production contoh:

```text
https://agripadibackendv2.petanitech.com
```

Base API:

```text
/api
```

### Health Check

| Method | Endpoint | Auth | Fungsi |
|---|---|---:|---|
| `GET` | `/health` | Tidak | Mengecek status backend |

### Auth

| Method | Endpoint | Auth | Fungsi |
|---|---|---:|---|
| `POST` | `/api/auth/register` | Tidak | Registrasi pengguna |
| `POST` | `/api/auth/login` | Tidak | Login pengguna |
| `POST` | `/api/auth/logout` | Ya | Logout pengguna |

### Profile

| Method | Endpoint | Auth | Fungsi |
|---|---|---:|---|
| `GET` | `/api/profile` | Ya | Ambil profil |
| `PATCH` | `/api/profile` | Ya | Update profil |
| `PATCH` | `/api/profile/password` | Ya | Ubah password |

### Upload

| Method | Endpoint | Auth | Fungsi |
|---|---|---:|---|
| `POST` | `/api/upload` | Ya | Upload gambar |
| `GET` | `/uploads/<nama-file>` | Tidak | Akses gambar upload |

### Conversations

| Method | Endpoint | Auth | Fungsi |
|---|---|---:|---|
| `GET` | `/api/conversations` | Ya | Ambil daftar percakapan |
| `POST` | `/api/conversations` | Ya | Buat atau ambil percakapan |
| `GET` | `/api/conversations/:conversation_id` | Ya | Ambil atau buat percakapan |
| `DELETE` | `/api/conversations/:conversation_id` | Ya | Hapus satu percakapan |
| `POST` | `/api/conversations/delete-many` | Ya | Hapus banyak percakapan |
| `POST` | `/api/conversations/delete-all` | Ya | Hapus semua percakapan |
| `DELETE` | `/api/conversations` | Ya | Hapus banyak percakapan |

### Messages

| Method | Endpoint | Auth | Fungsi |
|---|---|---:|---|
| `GET` | `/api/conversations/:conversation_id/messages` | Ya | Ambil pesan dalam percakapan |
| `POST` | `/api/conversations/:conversation_id/messages` | Ya | Kirim pesan user dan proses response assistant |

---

## Workflow Diagnosis

Alur umum:

1. Flutter menjalankan model CNN TFLite secara on-device.
2. Flutter mengirim hasil deteksi, gambar, dan pesan ke backend.
3. Backend menyimpan pesan pengguna.
4. Backend memproses workflow diagnosis.
5. Jika informasi belum cukup, backend meminta data tambahan:
   - gejala yang terlihat,
   - tingkat keparahan,
   - fase pertumbuhan padi.
6. Backend menjalankan rule engine.
7. Backend menentukan rekomendasi pengendalian.
8. LLM membantu menyusun narasi penjelasan.
9. Backend menyimpan dan mengirim pesan assistant ke frontend.

---

## Dataset Sistem Pakar

Dataset berada di:

```text
dataset/
```

| File | Fungsi |
|---|---|
| `pest.csv` | Master data hama padi |
| `pesticide.csv` | Master data pestisida / bahan aktif |
| `rules.csv` | Aturan rekomendasi pestisida |
| `severity.csv` | Data tingkat keparahan serangan |
| `growth_stage.csv` | Data fase pertumbuhan tanaman |
| `symptom_rule_simple.csv` | Data gejala sederhana untuk rule |
| `pesticide_application_timing_reference.csv` | Referensi waktu aplikasi pestisida |

Jaga konsistensi penamaan `pest_code`, `severity`, `growth_stage`, dan bahan aktif agar rule engine dapat bekerja dengan benar.

---

## Mode CNN

Default:

```env
CNN_MODE=on_device_flutter
```

Mode ini berarti CNN berjalan di Flutter, bukan di backend.

Jika suatu saat ingin memakai CNN server:

```env
CNN_MODE=server_api
CNN_URL=http://cnn-service:8180
```

Namun untuk versi aplikasi mobile ringan dan demo lomba, mode yang disarankan tetap:

```env
CNN_MODE=on_device_flutter
```

---

## Keamanan

Hal penting:

- Jangan commit `.env`.
- Jangan menulis password database di `config.yml`.
- Jangan menulis API key LLM di `config.yml`.
- Gunakan `JWT_SECRET` panjang dan acak.
- Gunakan HTTPS untuk domain backend.
- Batasi CORS pada domain production.
- Gunakan volume untuk menyimpan folder `/uploads`.
- Backup database secara berkala.
- Jangan membagikan log yang berisi token atau credential.

Membuat JWT secret:

```bash
openssl rand -base64 48
```

---

## Troubleshooting

### `database dsn is required`

Penyebab: `DATABASE_DSN` tidak masuk ke container.

Cek:

```bash
docker exec -it agripadi-backend-v2 printenv | grep DATABASE_DSN
```

Pastikan `.env` sejajar dengan `docker-compose.yml` dan service backend punya:

```yaml
env_file:
  - .env
```

---

### `jwt secret is required in production`

Penyebab: `JWT_SECRET` kosong.

Tambahkan:

```env
JWT_SECRET=isi_secret_panjang_minimal_32_karakter
```

Lalu restart:

```bash
docker compose up -d backend-v2
```

---

### Backend tidak bisa connect PostgreSQL

Jika backend berjalan di container, jangan gunakan `localhost` untuk host database.

Gunakan nama service:

```env
DATABASE_DSN=postgres://postgres:password_database@postgres:5432/agripadi?sslmode=disable
```

---

### Upload berhasil tetapi gambar tidak muncul

Pastikan route static aktif:

```text
/uploads
```

Pastikan volume upload tidak hilang:

```yaml
volumes:
  - uploads_data:/app/uploads
```

---

## Testing

Jalankan unit test:

```bash
go test ./...
```

Cek format:

```bash
gofmt -w .
```

Cek service berjalan:

```bash
curl http://localhost:8182/health
```

Jika memakai Docker:

```bash
docker compose ps
docker compose logs -f backend-v2
```

---

## Checklist Production

- [ ] `.env` tersedia di server.
- [ ] `DATABASE_DSN` benar.
- [ ] `JWT_SECRET` sudah aman.
- [ ] `LLM_API_KEY` sudah terisi jika memakai LLM.
- [ ] PostgreSQL sehat.
- [ ] Backend dapat diakses melalui `/health`.
- [ ] Domain backend memakai HTTPS.
- [ ] CORS hanya membuka domain yang diperlukan.
- [ ] Volume `/app/uploads` aktif.
- [ ] CI/CD tidak menimpa `.env`.
- [ ] Database sudah dibackup.

---

## Catatan

Backend ini merupakan bagian dari sistem AgriPadi AI untuk smart farming padi. Rekomendasi pestisida yang dihasilkan sistem harus digunakan sebagai alat bantu keputusan, dengan tetap memperhatikan label produk resmi, dosis, fase pertumbuhan tanaman, kondisi lapangan, ambang pengendalian, dan prinsip penggunaan pestisida yang bijak.
