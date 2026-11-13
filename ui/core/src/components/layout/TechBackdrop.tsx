/** 走线轨迹按 1440x900 的视口设计,负坐标让端点自然延伸到画面外。 */
const TRACES = [
  'M -40 168 H 236 L 292 112 H 486 L 534 160 H 742',
  'M 1480 372 H 1236 L 1180 316 H 1018 L 966 368 H 838',
  'M 96 940 V 792 L 152 736 H 402 L 448 782 H 634',
  'M 1480 112 H 1272 L 1216 172 H 1092',
  'M 1480 712 H 1310 L 1254 768 H 1096',
] as const

/** 走线拐点处的焊盘,呼吸动画错开相位避免整屏同步闪烁。 */
const NODES = [
  { x: 742, y: 160, r: 2.4 },
  { x: 534, y: 160, r: 1.8 },
  { x: 838, y: 368, r: 2.4 },
  { x: 1018, y: 316, r: 1.8 },
  { x: 634, y: 782, r: 2.4 },
  { x: 152, y: 736, r: 1.8 },
  { x: 1092, y: 172, r: 2.2 },
  { x: 1096, y: 768, r: 2.2 },
] as const

/**
 * 全主题的科技感底纹:光晕 + PCB 走线。
 * 纯装饰层,亮度由 index.css 中的 --backdrop-* 变量控制,避免抢走前景内容的注意力;
 * 走线只在深色主题出现,浅色主题只保留低对比度的光晕。
 */
export function TechBackdrop() {
  return (
    <div className="tech-backdrop" aria-hidden="true">
      <div className="tech-backdrop__glow tech-backdrop__glow--a" />
      <div className="tech-backdrop__glow tech-backdrop__glow--b" />

      <svg
        className="tech-backdrop__circuit"
        viewBox="0 0 1440 900"
        preserveAspectRatio="xMidYMid slice"
        fill="none"
      >
        <g className="tech-backdrop__trace">
          {TRACES.map((d) => (
            <path key={d} d={d} />
          ))}
        </g>
        <g>
          {TRACES.map((d, index) => (
            <path
              key={d}
              d={d}
              pathLength={100}
              className="tech-backdrop__pulse"
              style={{ animationDelay: `${index * -2.4}s` }}
            />
          ))}
        </g>
        <g className="tech-backdrop__node">
          {NODES.map((node, index) => (
            <circle
              key={`${node.x}-${node.y}`}
              cx={node.x}
              cy={node.y}
              r={node.r}
              style={{ animationDelay: `${index * 0.7}s` }}
            />
          ))}
        </g>
      </svg>

      <div className="tech-backdrop__vignette" />
    </div>
  )
}
