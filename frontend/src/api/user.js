import request from '@/utils/request'

// 修改当前用户密码
export function changePassword(oldPassword, newPassword) {
  return request.put('/auth/password', { old_password: oldPassword, new_password: newPassword })
}
