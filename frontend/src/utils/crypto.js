import CryptoJS from 'crypto-js'

/**
 * 使用 SHA-256 哈希密码
 * @param {string} password - 明文密码
 * @returns {string} - 哈希后的十六进制字符串 (64位)
 */
export function hashPassword(password) {
  return CryptoJS.SHA256(password).toString(CryptoJS.enc.Hex)
}

/**
 * 带盐值的密码哈希 (可选增强版)
 * @param {string} password - 明文密码
 * @param {string} username - 用户名作为盐值
 * @returns {string} - 哈希值
 */
export function hashPasswordWithSalt(password, username) {
  const salt = username.toLowerCase()
  return CryptoJS.SHA256(password + salt).toString(CryptoJS.enc.Hex)
}
