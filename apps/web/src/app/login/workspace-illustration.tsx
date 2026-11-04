export function WorkspaceIllustration() {
  return (
    <div className="login-illustration" aria-hidden="true">
      <svg viewBox="0 0 560 244" fill="none" focusable="false">
        <defs>
          <linearGradient
            id="database-face"
            x1="240"
            y1="75"
            x2="324"
            y2="170"
            gradientUnits="userSpaceOnUse"
          >
            <stop stopColor="#3474ff" />
            <stop offset="1" stopColor="#074ce0" />
          </linearGradient>
          <pattern
            id="workspace-grid"
            width="24"
            height="24"
            patternUnits="userSpaceOnUse"
          >
            <circle cx="1" cy="1" r="0.75" fill="#454c58" />
          </pattern>
          <filter
            id="database-shadow"
            x="200"
            y="40"
            width="160"
            height="174"
            filterUnits="userSpaceOnUse"
          >
            <feDropShadow
              dy="12"
              stdDeviation="12"
              floodColor="#074ce0"
              floodOpacity="0.2"
            />
          </filter>
        </defs>
        <rect width="560" height="244" fill="url(#workspace-grid)" />
        <ellipse
          cx="280"
          cy="123"
          rx="198"
          ry="79"
          stroke="#454c58"
          strokeOpacity="0.5"
        />
        <ellipse
          cx="280"
          cy="123"
          rx="132"
          ry="98"
          stroke="#454c58"
          strokeOpacity="0.35"
        />
        <path
          d="M132 84h48a24 24 0 0 1 24 24v15h32m88 0h32a24 24 0 0 0 24-24V84h48M132 163h48a24 24 0 0 0 24-24v-16m120 0h32a24 24 0 0 1 24 24v16h48"
          stroke="#697488"
          strokeWidth="1.2"
        />
        <circle cx="204" cy="123" r="3" fill="#a9b5cc" />
        <circle cx="356" cy="123" r="3" fill="#a9b5cc" />
        <g filter="url(#database-shadow)">
          <path
            d="M236 86c0-13 88-13 88 0v72c0 23-88 23-88 0V86Z"
            fill="url(#database-face)"
          />
          <ellipse cx="280" cy="86" rx="44" ry="16" fill="#5c91ff" />
          <ellipse
            cx="280"
            cy="86"
            rx="28"
            ry="8"
            stroke="#a7c4ff"
            strokeOpacity="0.8"
          />
          <path
            d="M236 110c0 22 88 22 88 0m-88 24c0 22 88 22 88 0"
            stroke="#8bb1ff"
            strokeOpacity="0.65"
          />
          <circle cx="307" cy="112" r="2" fill="#d8e5ff" />
          <circle cx="307" cy="136" r="2" fill="#d8e5ff" />
          <circle cx="307" cy="160" r="2" fill="#d8e5ff" />
        </g>
        <g stroke="#dde0e6">
          <rect x="94" y="62" width="44" height="44" rx="9" fill="#fcfcfd" />
          <rect x="422" y="62" width="44" height="44" rx="9" fill="#fcfcfd" />
          <rect x="94" y="142" width="44" height="44" rx="9" fill="#fcfcfd" />
          <rect x="422" y="142" width="44" height="44" rx="9" fill="#fcfcfd" />
        </g>
        <g
          stroke="#454c58"
          strokeWidth="1.4"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <rect x="106" y="74" width="20" height="20" rx="2" />
          <path d="M106 81h20m-13 0v13m-7-7h20" />
          <path d="m439 77-6 7 6 7m10-14 6 7-6 7m-6-16-2 18" />
          <rect x="106" y="154" width="7" height="7" rx="1" />
          <rect x="119" y="167" width="7" height="7" rx="1" />
          <path d="M109.5 161v9.5h9.5" />
          <path d="M434 159h20m-20 6h14m-14 6h17" />
        </g>
      </svg>
      <div className="login-illustration-caption">
        <span className="login-illustration-line" />
        <span>Schewise</span>
        <span className="login-illustration-line" />
      </div>
    </div>
  );
}
