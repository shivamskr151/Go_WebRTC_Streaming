import React from 'react'

interface Message {
  type: 'success' | 'error' | ''
  text: string
}

interface MessageToastProps {
  message: Message
}

export const MessageToast: React.FC<MessageToastProps> = ({ message }) => {
  if (!message || !message.text) {
    return null
  }

  const styles = message.type === 'success' 
    ? 'bg-green-100 text-green-800 border border-green-200'
    : 'bg-red-100 text-red-800 border border-red-200'

  return (
    <div className={`p-4 rounded-lg my-4 text-center ${styles}`}>
      {message.text}
    </div>
  )
}
