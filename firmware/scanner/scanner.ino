#include <BLEDevice.h>
#include <BLEScan.h>
#include <BLEAdvertisedDevice.h>
#include <WiFi.h>
#include <HTTPClient.h>
#include "config.h"

BLEScan* pBLEScan;
char scannerMac[18];

// Apple Find My network company ID
#define APPLE_COMPANY_ID 0x004C

// Check if device is iTag03 based on advertisement patterns
bool isITag03(BLEAdvertisedDevice& dev) {
  // iTag03 typically broadcasts with specific service UUIDs or manufacturer data
  // Check for Apple Find My service (0xFD6F) - common in Find My compatible tags
  if (dev.haveServiceUUID()) {
    String serviceUUID = dev.getServiceUUID().toString();
    // Apple Find My Network Service UUID: 0xFD6F
    if (serviceUUID.indexOf("fd6f") >= 0 ||
        serviceUUID.indexOf("FD6F") >= 0) {
      return true;
    }
  }

  // Check manufacturer data for Apple Find My
  if (dev.haveManufacturerData()) {
    String manuData = dev.getManufacturerData();
    if (manuData.length() >= 2) {
      uint16_t companyId = (uint8_t)manuData[0] | ((uint8_t)manuData[1] << 8);
      if (companyId == APPLE_COMPANY_ID) {
        // Apple device - likely Find My tag including iTag03
        return true;
      }
    }
  }

  // Check device name for iTag patterns
  if (dev.haveName()) {
    String name = dev.getName();
    // iTag03 often broadcasts with no name or generic name
    // Check if name contains "iTag" or is empty (common for tags)
    if (name.indexOf("iTag") >= 0 ||
        name.indexOf("ITAG") >= 0 ||
        name.length() == 0) {
      // Additional check: small devices with strong signal in close proximity
      // iTag03 is typically -40 to -60 dBm when very close
      if (dev.getRSSI() > -65) {
        return true;
      }
    }
  }

  return false;
}

// Get device type string
const char* getDeviceType(BLEAdvertisedDevice& dev) {
  if (isITag03(dev)) {
    return "iTag03";
  }

  // Check for other device types
  if (dev.haveManufacturerData()) {
    String manuData = dev.getManufacturerData();
    if (manuData.length() >= 2) {
      uint16_t companyId = (uint8_t)manuData[0] | ((uint8_t)manuData[1] << 8);

      switch (companyId) {
        case 0x004C: return "Apple";
        case 0x0075: return "Samsung";
        case 0x0006: return "Microsoft";
        case 0x0105: return "Google";
        default: break;
      }
    }
  }

  // Check service UUIDs for specific devices
  if (dev.haveServiceUUID()) {
    String serviceUUID = dev.getServiceUUID().toString();
    if (serviceUUID.indexOf("fe9f") >= 0 ||
        serviceUUID.indexOf("FE9F") >= 0) {
      return "Tile";
    }
    if (serviceUUID.indexOf("fd6f") >= 0 ||
        serviceUUID.indexOf("FD6F") >= 0) {
      return "FindMy_Tag";
    }
  }

  return "Unknown";
}

void setup() {
  Serial.begin(115200);
  Serial.println("🚀 เริ่มต้น BLE Scanner...");

  // เชื่อมต่อ WiFi
  Serial.printf("📡 กำลังเชื่อมต่อ WiFi: %s\n", ssid);
  WiFi.begin(ssid, password);
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }
  Serial.println("\n✅ เชื่อมต่อ WiFi สำเร็จ");

  WiFi.macAddress().toCharArray(scannerMac, 18);
  Serial.printf("📍 Scanner MAC: %s\n", scannerMac);

  // ตั้งค่า BLE
  BLEDevice::init("");
  pBLEScan = BLEDevice::getScan();
  pBLEScan->setActiveScan(true);
  pBLEScan->setInterval(100);
  pBLEScan->setWindow(99);

  Serial.println("✅ BLE Scanner พร้อมใช้งาน");
  Serial.println("📡 กำลังสแกนอุปกรณ์ Bluetooth...");
  Serial.println("🔍 Backend จะตรวจสอบว่าอุปกรณ์ไหนเป็น target device");
}

void loop() {
  BLEScanResults* devices = pBLEScan->start(scanDuration, false);

  Serial.printf("📡 พบอุปกรณ์ %d เครื่อง\n", devices->getCount());

  for (int i = 0; i < devices->getCount(); i++) {
    BLEAdvertisedDevice dev = devices->getDevice(i);
    int rssi = dev.getRSSI();

    // Only send if device is close enough (within ~10 meters)
    if (rssi < rssiThreshold) {
      continue; // Skip devices that are too far
    }

    char mac[18];
    dev.getAddress().toString().toCharArray(mac, 18);

    // Get device type
    const char* deviceType = getDeviceType(dev);

    // Check if it's an iTag03
    bool itag03Detected = isITag03(dev);

    HTTPClient http;
    http.begin(backendUrl);
    http.addHeader("Content-Type", "application/json");

    // ส่งข้อมูลทุกอุปกรณ์ไป backend โดยไม่ต้องเช็ค target device
    // Backend จะเป็นคนตัดสินใจว่าอันไหนเป็น target device
    char json[512];
    sprintf(json, "{\"scanner_mac\":\"%s\",\"mac_address\":\"%s\",\"rssi\":%d,\"device_type\":\"%s\",\"itag03\":%s}",
            scannerMac, mac, rssi, deviceType,
            itag03Detected ? "true" : "false");

    int httpCode = http.POST(json);

    Serial.printf("📡 ส่งข้อมูล: %s | RSSI: %d | Type: %s (HTTP %d)\n",
                  mac, rssi, deviceType, httpCode);

    http.end();
  }

  pBLEScan->clearResults();
  delay(scanInterval);
}
