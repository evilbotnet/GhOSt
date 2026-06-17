<script lang="ts">
  // Hand-authored 24x24 stroke icons — one path string per name keeps the
  // whole set a few KB and avoids an icon-library dependency.
  const PATHS: Record<string, string> = {
    launcher:
      'M4 4h4v4H4zM10 4h4v4h-4zM16 4h4v4h-4zM4 10h4v4H4zM10 10h4v4h-4zM16 10h4v4h-4zM4 16h4v4H4zM10 16h4v4h-4zM16 16h4v4h-4z',
    files:
      'M3 6a2 2 0 0 1 2-2h4l2 3h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z',
    folder:
      'M3 6a2 2 0 0 1 2-2h4l2 3h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z',
    file: 'M6 2h8l4 4v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zM14 2v5h5',
    terminal: 'M4 5h16a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1zM7 9l3 3-3 3M12 15h5',
    editor: 'M6 2h8l4 4v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zM14 2v5h5M8 13h8M8 17h5',
    settings:
      'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.2 2.2M16.9 16.9l2.2 2.2M19.1 4.9l-2.2 2.2M7.1 16.9l-2.2 2.2',
    browser:
      'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM3 12h18M12 3c2.5 2.4 4 5.6 4 9s-1.5 6.6-4 9c-2.5-2.4-4-5.6-4-9s1.5-6.6 4-9z',
    office:
      'M6 2h8l4 4v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2zM14 2v5h5M8 12h8M8 15h8M8 18h4',
    info: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM12 11v5M12 8v.01',
    close: 'M5 5l14 14M19 5L5 19',
    minimize: 'M5 12h14',
    maximize: 'M5 5h14v14H5z',
    restore: 'M8 8h12v12H8zM8 8V4h12v12h-4',
    search: 'M10 4a6 6 0 1 0 0 12 6 6 0 0 0 0-12zM14.5 14.5L20 20',
    'chevron-up': 'M6 14l6-6 6 6',
    'arrow-up': 'M12 19V5M5 12l7-7 7 7',
    'arrow-left': 'M19 12H5M12 5l-7 7 7 7',
    home: 'M3 11l9-8 9 8M5 9v11h5v-6h4v6h5V9',
    plus: 'M12 5v14M5 12h14',
    refresh: 'M20 8A8 8 0 1 0 20 16M20 3v5h-5',
    trash: 'M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13M10 11v6M14 11v6',
    image: 'M4 4h16v16H4zM4 15l5-5 4 4 3-3 4 4M9 9h.01',
    wifi: 'M2 9c5.5-5.3 14.5-5.3 20 0M5.5 12.5c3.6-3.4 9.4-3.4 13 0M9 16c1.7-1.6 4.3-1.6 6 0M12 19.5v.01',
    battery: 'M3 8h15a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V9a1 1 0 0 1 1-1zM22 11v2M4.5 10.5h6v3h-6z',
    volume: 'M4 9v6h4l5 4V5L8 9zM16 9a4 4 0 0 1 0 6M18.5 6.5a8 8 0 0 1 0 11',
    power: 'M12 3v8M6.2 6.2a8 8 0 1 0 11.6 0',
    lock: 'M6 11h12v9H6zM8 11V7a4 4 0 0 1 8 0v4',
    save: 'M5 3h11l3 3v15H5zM8 3v5h7V3M8 21v-7h8v7',
    camera:
      'M3 8a2 2 0 0 1 2-2h2l2-3h6l2 3h2a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2zM12 17a4 4 0 1 0 0-7.5 4 4 0 0 0 0 7.5z',
    chart: 'M4 20V4M4 20h16M8 20v-6M13 20V9M18 20v-9',
    puzzle:
      'M10 3h4v3a2 2 0 0 0 4 0V3h0v4h3a0 0 0 0 1 0 0v0a2 2 0 0 1 0 4h0v4h-3a2 2 0 0 0 0 4h3v3H4V11h3a2 2 0 0 0 0-4H4V3z',
  };

  let {
    name,
    size = 16,
    stroke = 1.7,
  }: { name: string; size?: number; stroke?: number } = $props();
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width={stroke}
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"
>
  <path d={PATHS[name] ?? PATHS.file} />
</svg>
