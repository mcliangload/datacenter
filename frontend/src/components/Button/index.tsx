import React from 'react'

interface ButtonProps {
  children: React.ReactNode
  onClick?: () => void
  type?: 'primary' | 'default'
}

const Button: React.FC<ButtonProps> = ({ children, onClick, type = 'primary' }) => {
  return (
    <button
      onClick={onClick}
      style={{
        padding: '8px 16px',
        backgroundColor: type === 'primary' ? '#1890ff' : '#fff',
        color: type === 'primary' ? '#fff' : '#000',
        border: '1px solid #d9d9d9',
        borderRadius: '4px',
        cursor: 'pointer',
      }}
    >
      {children}
    </button>
  )
}

export default Button