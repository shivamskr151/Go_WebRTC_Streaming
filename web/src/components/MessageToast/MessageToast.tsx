import React, { useEffect, useState } from 'react'

interface Message {
  type: 'success' | 'error' | ''
  text: string
}

interface MessageToastProps {
  message: Message
}

export const MessageToast: React.FC<MessageToastProps> = ({ message }) => {
  const [isVisible, setIsVisible] = useState(false)

  useEffect(() => {
    if (message && message.text) {
      setIsVisible(true)
      const timer = setTimeout(() => setIsVisible(false), 3000)
      return () => clearTimeout(timer)
    }
  }, [message])

  if (!message || !message.text || !isVisible) {
    return null
  }

  const styles = message.type === 'success' 
    ? 'bg-gradient-to-r from-green-50 to-emerald-50 text-green-800 border-2 border-green-300 shadow-lg shadow-green-200/50'
    : 'bg-gradient-to-r from-red-50 to-rose-50 text-red-800 border-2 border-red-300 shadow-lg shadow-red-200/50'

  const icon = message.type === 'success' ? '✅' : '❌'

  return (
    <div className={`fixed top-4 left-1/2 transform -translate-x-1/2 z-50 px-6 py-4 rounded-xl my-4 text-center font-semibold backdrop-blur-sm animate-slide-down max-w-md w-[90%] md:w-auto ${styles}`}>
      <div className="flex items-center justify-center gap-2">
        <span className="text-xl">{icon}</span>
        <span>{message.text}</span>
      </div>
    </div>
  )
}
