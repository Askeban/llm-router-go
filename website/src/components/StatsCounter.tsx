import { useEffect, useState } from 'react'

interface StatsCounterProps {
  icon: React.ReactNode
  value: number
  label: string
  suffix?: string
  duration?: number
}

export default function StatsCounter({ icon, value, label, suffix = '', duration = 2000 }: StatsCounterProps) {
  const [count, setCount] = useState(0)
  const [isVisible, setIsVisible] = useState(false)

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsVisible(true)
        }
      },
      { threshold: 0.1 }
    )

    const element = document.getElementById(`stats-${label.replace(/\s+/g, '-').toLowerCase()}`)
    if (element) observer.observe(element)

    return () => observer.disconnect()
  }, [label])

  useEffect(() => {
    if (!isVisible) return

    let start = 0
    const end = value
    const increment = end / (duration / 16) // 60 FPS
    
    const timer = setInterval(() => {
      start += increment
      if (start >= end) {
        setCount(end)
        clearInterval(timer)
      } else {
        setCount(Math.floor(start))
      }
    }, 16)

    return () => clearInterval(timer)
  }, [isVisible, value, duration])

  return (
    <div 
      id={`stats-${label.replace(/\s+/g, '-').toLowerCase()}`}
      className="text-center animate-slide-up"
    >
      <div className="flex justify-center mb-3">
        {icon}
      </div>
      <div className="text-2xl md:text-3xl font-bold text-gray-900 mb-1 animate-counter">
        {count.toLocaleString()}{suffix}
      </div>
      <div className="text-sm text-gray-600 font-medium">
        {label}
      </div>
    </div>
  )
}