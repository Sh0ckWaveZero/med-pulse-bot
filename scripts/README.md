# Database Migration Scripts

สคริปต์สำหรับ migrate PocketBase database เพื่อเพิ่มฟิลด์ target device detection

## 📋 Overview

Migration นี้เพิ่มฟิลด์ 2 ฟิลด์ใน `employee_detections` collection:
- `is_target_device` (Boolean) - ระบุว่าเป็นอุปกรณ์เป้าหมายหรือไม่
- `device_name` (Text, max 255) - ชื่อของอุปกรณ์เป้าหมาย (เช่น "MSL AirPods Pro")

## 🚀 วิธีใช้งาน

### วิธีที่ 1: Bash Script (แนะนำ)

```bash
# สคริปต์จะอ่าน .env file อัตโนมัติ
chmod +x scripts/migrate_add_target_device_fields.sh
./scripts/migrate_add_target_device_fields.sh
```

**ข้อกำหนด .env file:**
```bash
POCKETBASE_URL=http://192.168.100.100:8090
POCKETBASE_TOKEN=your_auth_token_here
```

### วิธีที่ 2: Go Script

```bash
# สคริปต์จะอ่าน .env file อัตโนมัติ
go run scripts/migrate/main.go
```

### วิธีที่ 3: Make Command

```bash
# รัน Bash script
make migrate-db

# หรือรัน Go script
make migrate-db-go
```

## 📦 Dependencies

### Bash Script
- `curl` - HTTP client
- `jq` - JSON processor
- `grep` - Text search

ติดตั้งบน macOS:
```bash
brew install jq
```

ติดตั้งบน Ubuntu/Debian:
```bash
sudo apt-get install jq curl
```

### Go Script
- Go 1.21+
- ไม่ต้องติดตั้ง dependencies เพิ่มเติม (ใช้ standard library)

## 🔐 Authentication

สคริปต์ใช้ **POCKETBASE_TOKEN** จากไฟล์ .env เพื่อแก้ไข collection schema:

### วิธีหา Token:
1. เข้า PocketBase Admin UI: `http://192.168.100.100:8090/_/`
2. Login ด้วย admin account
3. เปิด Developer Tools (F12) → Network tab
4. Refresh หน้า → ดู Request headers
5. คัดลอก token จาก Authorization header

### Environment Variables (.env):
```bash
POCKETBASE_URL=http://192.168.100.100:8090
POCKETBASE_TOKEN=eyJhbGci...  # Auth token from PocketBase
```

## ✅ Verification

หลังรันสคริปต์เสร็จ ตรวจสอบว่า migration สำเร็จ:

### 1. ผ่าน API
```bash
curl http://192.168.100.135:8090/api/collections/employee_detections | jq '.schema'
```

### 2. ผ่าน Admin UI
1. เข้า `http://192.168.100.135:8090/_/`
2. ไปที่ Collections → employee_detections
3. ตรวจสอบว่ามีฟิลด์ `is_target_device` และ `device_name`

### 3. ทดสอบการบันทึกข้อมูล
```bash
# ส่ง test detection
curl -X POST http://192.168.100.135:8080/api/detect \
  -H "Content-Type: application/json" \
  -d '{
    "scanner_mac": "aa:bb:cc:dd:ee:ff",
    "mac_address": "11:22:33:44:55:66",
    "rssi": -55,
    "device_type": "Apple",
    "itag03": false,
    "target_device": true,
    "device_name": "MSL AirPods Pro"
  }'
```

## 🔄 Rollback

ถ้าต้องการ rollback migration:

### ผ่าน Admin UI (แนะนำ):
1. เข้า Collections → employee_detections
2. ลบฟิลด์ `is_target_device` และ `device_name`
3. Save

### ผ่าน API:
```bash
# ดู collection ID
COLLECTION_ID=$(curl -s http://192.168.100.135:8090/api/collections/employee_detections | jq -r '.id')

# ลบฟิลด์ (ต้องมี admin token)
# ... (implement rollback script if needed)
```

## 📝 Troubleshooting

### Error: "Cannot connect to PocketBase"
```bash
# ตรวจสอบว่า PocketBase running
docker-compose ps

# Restart PocketBase
docker-compose restart
```

### Error: "Authentication failed"
```bash
# ตรวจสอบ credentials
echo $POCKETBASE_ADMIN_EMAIL
echo $POCKETBASE_ADMIN_PASSWORD

# ลอง login ผ่าน Admin UI
open http://192.168.100.135:8090/_/
```

### Error: "Field already exists"
- ไม่เป็นไร! สคริปต์จะข้ามฟิลด์ที่มีอยู่แล้ว
- Migration ยังคงดำเนินการต่อ

### Error: "jq: command not found"
```bash
# ติดตั้ง jq
brew install jq          # macOS
sudo apt install jq      # Ubuntu
```

## 📚 Additional Resources

- [PocketBase Collections API](https://pocketbase.io/docs/api-collections/)
- [PocketBase Schema Fields](https://pocketbase.io/docs/collections/#schema-fields)
- [Project POCKETBASE_MIGRATION.md](../POCKETBASE_MIGRATION.md)

## 🎯 Next Steps

หลัง migrate เสร็จ:

1. ✅ Restart Backend API
   ```bash
   docker-compose restart app
   ```

2. ✅ Upload Firmware to ESP32
   - เปิด Arduino IDE หรือ PlatformIO
   - Upload code ไปยัง ESP32

3. ✅ Test System
   - เปิด Serial Monitor (baud rate: 115200)
   - ดู logs เมื่อเจอ target device
   - ตรวจสอบ backend logs: `docker-compose logs -f app`

4. ✅ Configure Target Devices
   - แก้ไข `firmware/scanner/config.h`
   - เพิ่ม MAC addresses และ UUIDs
   - Upload firmware ใหม่
