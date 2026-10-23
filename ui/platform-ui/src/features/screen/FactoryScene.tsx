import { useEffect, useRef, useState } from 'react'
import * as THREE from 'three'
import { OrbitControls } from 'three/addons/controls/OrbitControls.js'
import type { DeviceStatus } from '@/api/types'
import { DEVICE_STATUS, PROTOCOLS, UNKNOWN_STATUS } from '@/lib/constants'
import { cn } from '@/lib/cn'
import {
  SCREEN_COLORS,
  STATUS_COLOR,
  UNKNOWN_STATUS_COLOR,
  ZONE_ACCENT,
  ZONE_DEPTH,
  ZONE_ORDER,
  ZONE_TEXT,
  ZONE_WIDTH,
  ZONE_X,
  type FactoryDevice,
} from './layout'

interface FactorySceneProps {
  devices: FactoryDevice[]
  selectedId: string | null
  onSelect: (id: string | null) => void
}

interface DeviceObjects {
  bodyMaterial: THREE.MeshStandardMaterial
  capMaterial: THREE.MeshBasicMaterial
  floatMaterial: THREE.MeshBasicMaterial
  beamMaterial: THREE.MeshBasicMaterial
  ring: THREE.Mesh<THREE.RingGeometry, THREE.MeshBasicMaterial>
  floater: THREE.Mesh
  height: number
  status: DeviceStatus
}

const RUNNING: DeviceStatus[] = ['collecting', 'collectingError']

export function FactoryScene({ devices, selectedId, onSelect }: FactorySceneProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const labelsRef = useRef(new Map<string, HTMLButtonElement>())
  const zoneLabelsRef = useRef(new Map<string, HTMLSpanElement>())
  const objectsRef = useRef(new Map<string, DeviceObjects>())
  const devicesRef = useRef(devices)
  const selectedIdRef = useRef(selectedId)
  const onSelectRef = useRef(onSelect)
  const [failed, setFailed] = useState(false)

  // 场景是命令式重建的,通过 ref 读取最新状态,避免把轮询数据放进场景依赖。
  useEffect(() => {
    devicesRef.current = devices
  }, [devices])
  useEffect(() => {
    selectedIdRef.current = selectedId
  }, [selectedId])
  useEffect(() => {
    onSelectRef.current = onSelect
  }, [onSelect])

  const layoutKey = devices
    .map((item) => `${item.device.id}:${item.device.deviceType}:${item.x}:${item.z}`)
    .join('|')

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    let renderer: THREE.WebGLRenderer
    try {
      renderer = new THREE.WebGLRenderer({ antialias: true, alpha: false })
    } catch {
      setFailed(true)
      return
    }
    setFailed(false)

    const canvas = renderer.domElement
    canvas.style.display = 'block'
    canvas.style.width = '100%'
    canvas.style.height = '100%'
    canvas.style.touchAction = 'none'
    container.appendChild(canvas)

    const scene = new THREE.Scene()
    scene.background = new THREE.Color(SCREEN_COLORS.background)
    scene.fog = new THREE.Fog(SCREEN_COLORS.background, 28, 62)

    const camera = new THREE.PerspectiveCamera(46, 1, 0.1, 200)
    camera.position.set(0, 14.5, 21.5)

    const controls = new OrbitControls(camera, canvas)
    controls.target.set(0, 0.7, 0)
    controls.enableDamping = true
    controls.dampingFactor = 0.08
    controls.enablePan = false
    controls.minDistance = 14
    controls.maxDistance = 46
    controls.minPolarAngle = 0.35
    controls.maxPolarAngle = 1.36
    controls.rotateSpeed = 0.7
    controls.zoomSpeed = 0.8
    controls.autoRotate = !window.matchMedia('(prefers-reduced-motion: reduce)').matches
    controls.autoRotateSpeed = 0.45
    controls.update()

    scene.add(new THREE.AmbientLight(0x9ec9d6, 0.55))
    scene.add(new THREE.HemisphereLight(0x9fd8e8, 0x08141a, 0.85))
    const keyLight = new THREE.DirectionalLight(0xffffff, 1.15)
    keyLight.position.set(14, 22, 12)
    scene.add(keyLight)
    const rimLight = new THREE.DirectionalLight(0x38bdf8, 0.5)
    rimLight.position.set(-16, 10, -12)
    scene.add(rimLight)

    /* ---------- 地面与分区 ---------- */

    const floor = new THREE.Mesh(
      new THREE.PlaneGeometry(44, 28),
      new THREE.MeshStandardMaterial({
        color: SCREEN_COLORS.floor,
        roughness: 0.95,
        metalness: 0.05,
      }),
    )
    floor.rotation.x = -Math.PI / 2
    floor.position.y = -0.02
    scene.add(floor)

    const grid = new THREE.GridHelper(44, 44, SCREEN_COLORS.grid, SCREEN_COLORS.grid)
    grid.position.y = 0.01
    const gridMaterial = grid.material as THREE.LineBasicMaterial
    gridMaterial.transparent = true
    gridMaterial.opacity = 0.32
    scene.add(grid)

    for (const type of ZONE_ORDER) {
      const accent = ZONE_ACCENT[type]
      const geometry = new THREE.BoxGeometry(ZONE_WIDTH, 0.18, ZONE_DEPTH)
      const platform = new THREE.Mesh(
        geometry,
        new THREE.MeshStandardMaterial({
          color: SCREEN_COLORS.platform,
          emissive: accent,
          emissiveIntensity: 0.05,
          roughness: 0.7,
          metalness: 0.3,
          transparent: true,
          opacity: 0.92,
        }),
      )
      platform.position.set(ZONE_X[type], 0.09, 0)
      scene.add(platform)

      const edges = new THREE.LineSegments(
        new THREE.EdgesGeometry(geometry),
        new THREE.LineBasicMaterial({ color: accent, transparent: true, opacity: 0.45 }),
      )
      edges.position.copy(platform.position)
      scene.add(edges)
    }

    /* ---------- 环境粒子 ---------- */

    const particleCount = 220
    const positions = new Float32Array(particleCount * 3)
    let seed = 20261023
    const random = () => {
      seed = (seed * 1664525 + 1013904223) >>> 0
      return seed / 0xffffffff
    }
    for (let i = 0; i < particleCount; i += 1) {
      positions[i * 3] = (random() - 0.5) * 42
      positions[i * 3 + 1] = 0.4 + random() * 7
      positions[i * 3 + 2] = (random() - 0.5) * 26
    }
    const particleGeometry = new THREE.BufferGeometry()
    particleGeometry.setAttribute('position', new THREE.BufferAttribute(positions, 3))
    const particles = new THREE.Points(
      particleGeometry,
      new THREE.PointsMaterial({
        color: 0x2a7f8c,
        size: 0.055,
        transparent: true,
        opacity: 0.55,
        depthWrite: false,
      }),
    )
    scene.add(particles)

    /* ---------- 设备 ---------- */

    const objects = new Map<string, DeviceObjects>()
    const pickMeshes: THREE.Mesh[] = []

    for (const item of devicesRef.current) {
      const status = item.device.collectStatus
      const color = STATUS_COLOR[status] ?? UNKNOWN_STATUS_COLOR
      const group = new THREE.Group()
      group.position.set(item.x, 0, item.z)

      const base = new THREE.Mesh(
        new THREE.CylinderGeometry(0.62, 0.72, 0.12, 6),
        new THREE.MeshStandardMaterial({ color: 0x0e2530, metalness: 0.6, roughness: 0.4 }),
      )
      base.position.y = 0.06
      group.add(base)

      const bodyMaterial = new THREE.MeshStandardMaterial({
        color: SCREEN_COLORS.body,
        emissive: color,
        emissiveIntensity: RUNNING.includes(status) ? 0.45 : 0.16,
        metalness: 0.45,
        roughness: 0.35,
      })
      const body = new THREE.Mesh(new THREE.BoxGeometry(0.72, item.height, 0.72), bodyMaterial)
      body.position.y = 0.18 + item.height / 2
      body.userData.deviceId = item.device.id
      group.add(body)
      pickMeshes.push(body)

      const capMaterial = new THREE.MeshBasicMaterial({ color })
      const cap = new THREE.Mesh(new THREE.BoxGeometry(0.82, 0.07, 0.82), capMaterial)
      cap.position.y = 0.18 + item.height + 0.035
      group.add(cap)

      const floatMaterial = new THREE.MeshBasicMaterial({
        color,
        transparent: true,
        opacity: 0.85,
      })
      const floater = new THREE.Mesh(new THREE.OctahedronGeometry(0.13), floatMaterial)
      floater.position.y = 0.18 + item.height + 0.55
      group.add(floater)

      const beamMaterial = new THREE.MeshBasicMaterial({
        color,
        transparent: true,
        opacity: 0.28,
      })
      const beam = new THREE.Mesh(new THREE.CylinderGeometry(0.012, 0.012, 0.45, 6), beamMaterial)
      beam.position.y = 0.18 + item.height + 0.3
      group.add(beam)

      const ring = new THREE.Mesh(
        new THREE.RingGeometry(0.62, 0.78, 40),
        new THREE.MeshBasicMaterial({
          color,
          transparent: true,
          opacity: 0.85,
          side: THREE.DoubleSide,
        }),
      )
      ring.rotation.x = -Math.PI / 2
      ring.position.y = 0.17
      ring.visible = false
      group.add(ring)

      scene.add(group)
      objects.set(item.device.id, {
        bodyMaterial,
        capMaterial,
        floatMaterial,
        beamMaterial,
        ring,
        floater,
        height: item.height,
        status,
      })
    }
    objectsRef.current = objects

    /* ---------- 交互 ---------- */

    const raycaster = new THREE.Raycaster()
    const pointer = new THREE.Vector2()
    let hoveredId: string | null = null
    let pointerDown: { x: number; y: number } | null = null

    const pick = (event: PointerEvent): string | null => {
      const rect = canvas.getBoundingClientRect()
      pointer.x = ((event.clientX - rect.left) / rect.width) * 2 - 1
      pointer.y = -((event.clientY - rect.top) / rect.height) * 2 + 1
      raycaster.setFromCamera(pointer, camera)
      const hit = raycaster.intersectObjects(pickMeshes, false)[0]
      return (hit?.object.userData.deviceId as string | undefined) ?? null
    }

    const handlePointerMove = (event: PointerEvent) => {
      const id = pick(event)
      if (id === hoveredId) return
      hoveredId = id
      canvas.style.cursor = id ? 'pointer' : 'grab'
    }
    const handlePointerDown = (event: PointerEvent) => {
      pointerDown = { x: event.clientX, y: event.clientY }
    }
    const handlePointerUp = (event: PointerEvent) => {
      if (!pointerDown) return
      const moved = Math.hypot(event.clientX - pointerDown.x, event.clientY - pointerDown.y)
      pointerDown = null
      // 拖拽旋转后不触发选中。
      if (moved > 6) return
      onSelectRef.current(pick(event))
    }
    const handlePointerLeave = () => {
      hoveredId = null
      canvas.style.cursor = 'grab'
    }

    canvas.addEventListener('pointermove', handlePointerMove)
    canvas.addEventListener('pointerdown', handlePointerDown)
    canvas.addEventListener('pointerup', handlePointerUp)
    canvas.addEventListener('pointerleave', handlePointerLeave)

    /* ---------- 尺寸 ---------- */

    let viewWidth = Math.max(container.clientWidth, 1)
    let viewHeight = Math.max(container.clientHeight, 1)

    const resize = () => {
      viewWidth = Math.max(container.clientWidth, 1)
      viewHeight = Math.max(container.clientHeight, 1)
      camera.aspect = viewWidth / viewHeight
      // 窄屏横向视野更小,放大 fov 让三个分区同时可见。
      camera.fov = camera.aspect < 1 ? 60 : 46
      camera.updateProjectionMatrix()
      renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
      renderer.setSize(viewWidth, viewHeight, false)
    }
    resize()

    const observer = new ResizeObserver(resize)
    observer.observe(container)

    /* ---------- 标签投影 ---------- */

    const labelTargets = devicesRef.current.map((item) => ({
      id: item.device.id,
      position: new THREE.Vector3(item.x, 0.18 + item.height + 1.15, item.z),
    }))
    const zoneTargets = ZONE_ORDER.map((type) => ({
      type,
      position: new THREE.Vector3(ZONE_X[type], 0.25, ZONE_DEPTH / 2 + 1.1),
    }))

    const projected = new THREE.Vector3()
    const cameraDirection = new THREE.Vector3()
    const toPoint = new THREE.Vector3()

    const placeLabel = (element: HTMLElement | undefined, position: THREE.Vector3) => {
      if (!element) return
      camera.getWorldDirection(cameraDirection)
      toPoint.copy(position).sub(camera.position)
      projected.copy(position).project(camera)
      const visible =
        cameraDirection.dot(toPoint) > 0 && projected.z > -1 && projected.z < 1
      if (!visible) {
        element.style.opacity = '0'
        return
      }
      const x = (projected.x * 0.5 + 0.5) * viewWidth
      const y = (-projected.y * 0.5 + 0.5) * viewHeight
      element.style.opacity = '1'
      element.style.transform = `translate3d(${x.toFixed(1)}px, ${y.toFixed(1)}px, 0) translate(-50%, -50%)`
    }

    /* ---------- 渲染循环 ---------- */

    let frame = 0
    let last = performance.now()

    const animate = () => {
      frame = requestAnimationFrame(animate)
      const now = performance.now()
      const delta = Math.min((now - last) / 1000, 0.1)
      last = now
      const elapsed = now / 1000

      controls.update()
      particles.rotation.y += delta * 0.012

      for (const [id, object] of objects) {
        const running = RUNNING.includes(object.status)
        const phase = object.height * 4
        object.bodyMaterial.emissiveIntensity = running
          ? 0.38 + Math.sin(elapsed * 2 + phase) * 0.12
          : 0.16
        object.floater.rotation.y += delta * 1.1
        object.floater.position.y =
          0.18 + object.height + 0.55 + Math.sin(elapsed * 1.8 + phase) * 0.06

        if (id === selectedIdRef.current) {
          object.ring.visible = true
          object.ring.scale.setScalar(1 + Math.sin(elapsed * 3) * 0.06)
          object.ring.material.opacity = 0.65 + Math.sin(elapsed * 3) * 0.2
        } else {
          object.ring.visible = false
        }
      }

      for (const target of labelTargets) {
        placeLabel(labelsRef.current.get(target.id), target.position)
      }
      for (const target of zoneTargets) {
        placeLabel(zoneLabelsRef.current.get(target.type), target.position)
      }

      renderer.render(scene, camera)
    }
    animate()

    return () => {
      cancelAnimationFrame(frame)
      observer.disconnect()
      canvas.removeEventListener('pointermove', handlePointerMove)
      canvas.removeEventListener('pointerdown', handlePointerDown)
      canvas.removeEventListener('pointerup', handlePointerUp)
      canvas.removeEventListener('pointerleave', handlePointerLeave)
      controls.dispose()
      scene.traverse((object) => {
        if (
          object instanceof THREE.Mesh ||
          object instanceof THREE.LineSegments ||
          object instanceof THREE.Points
        ) {
          object.geometry.dispose()
          const material = object.material
          if (Array.isArray(material)) material.forEach((entry) => entry.dispose())
          else material.dispose()
        }
      })
      renderer.dispose()
      renderer.forceContextLoss()
      canvas.remove()
      objectsRef.current = new Map()
    }
  }, [layoutKey])

  // 只刷新材质状态,不重建场景,避免相机视角被轮询重置。
  useEffect(() => {
    for (const item of devices) {
      const object = objectsRef.current.get(item.device.id)
      if (!object) continue
      const status = item.device.collectStatus
      const color = STATUS_COLOR[status] ?? UNKNOWN_STATUS_COLOR
      object.status = status
      object.bodyMaterial.emissive.setHex(color)
      object.bodyMaterial.emissiveIntensity = RUNNING.includes(status) ? 0.45 : 0.16
      object.capMaterial.color.setHex(color)
      object.floatMaterial.color.setHex(color)
      object.floatMaterial.opacity = RUNNING.includes(status) ? 0.85 : 0.35
      object.beamMaterial.color.setHex(color)
      object.beamMaterial.opacity = RUNNING.includes(status) ? 0.28 : 0.12
      object.ring.material.color.setHex(color)
      object.ring.visible = item.device.id === selectedId
    }
  }, [devices, selectedId, layoutKey])

  if (failed) {
    return (
      <div className="flex h-full min-h-80 flex-col items-center justify-center gap-2 px-6 text-center">
        <p className="text-sm font-medium text-fg">无法渲染 3D 场景</p>
        <p className="text-xs text-muted">当前浏览器或显卡不支持 WebGL,请更换浏览器后重试。</p>
      </div>
    )
  }

  return (
    <div ref={containerRef} className="absolute inset-0">
      <div className="pointer-events-none absolute inset-0 z-10">
        {ZONE_ORDER.map((type) => (
          <span
            key={type}
            ref={(element) => {
              if (element) zoneLabelsRef.current.set(type, element)
              else zoneLabelsRef.current.delete(type)
            }}
            className={cn(
              'absolute top-0 left-0 rounded-full border border-border bg-bg/60 px-2.5 py-1 text-[10px] font-medium tracking-wider whitespace-nowrap uppercase backdrop-blur-sm',
              ZONE_TEXT[type],
            )}
            style={{ opacity: 0, willChange: 'transform' }}
          >
            {PROTOCOLS[type].label}
          </span>
        ))}

        {devices.map(({ device }) => {
          const meta = DEVICE_STATUS[device.collectStatus] ?? UNKNOWN_STATUS
          return (
            <button
              key={device.id}
              ref={(element) => {
                if (element) labelsRef.current.set(device.id, element)
                else labelsRef.current.delete(device.id)
              }}
              type="button"
              onClick={() => onSelect(device.id)}
              className={cn(
                'pointer-events-auto absolute top-0 left-0 inline-flex max-w-36 items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] whitespace-nowrap backdrop-blur-sm transition-colors',
                device.id === selectedId
                  ? 'border-primary/60 bg-primary-soft text-primary'
                  : 'border-border bg-bg/70 text-fg hover:border-border-strong hover:bg-surface',
              )}
              style={{ opacity: 0, willChange: 'transform' }}
            >
              <span className={cn('size-1.5 shrink-0 rounded-full', meta.dotClassName)} />
              <span className="truncate">{device.name}</span>
            </button>
          )
        })}
      </div>
    </div>
  )
}
