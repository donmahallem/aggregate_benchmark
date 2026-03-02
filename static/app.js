function colorFor(key) {
  let h = 0;
  for (let i = 0; i < key.length; i++) h = (h * 31 + key.charCodeAt(i)) >>> 0;
  return "#" + (h & 0xffffff).toString(16).padStart(6, "0");
}

function durationToMs(dur) {
  const m = dur.match(/^(\d+(?:\.\d+)?)(ns|us|µs|ms|s|m|h)$/);
  if (!m) return null;
  const v = parseFloat(m[1]);
  switch (m[2]) {
    case "ns":
      return v / 1_000_000;
    case "us":
    case "µs":
      return v / 1_000;
    case "ms":
      return v;
    case "s":
      return v * 1_000;
    case "m":
      return v * 60_000;
    case "h":
      return v * 3_600_000;
  }
  return null;
}

function formatMs(ms) {
  if (ms === null || ms === undefined) return "N/A";
  if (ms < 0.001) return (ms * 1_000_000).toFixed(1) + " ps";
  if (ms < 1) return (ms * 1_000).toFixed(2) + " µs";
  if (ms < 1_000) return ms.toFixed(3) + " ms";
  return (ms / 1_000).toFixed(3) + " s";
}

(async () => {
  const statusEl = document.getElementById("status");
  const gridEl = document.getElementById("grid");
  const metaEl = document.getElementById("meta");

  let history;
  let meta = {};
  try {
    const res = await fetch("./data.json");
    if (!res.ok) throw new Error(`HTTP ${res.status} ${res.statusText}`);
    const data = await res.json();
    history = data.history || [];
    meta = data.meta || {};
  } catch (e) {
    statusEl.textContent = `Could not load data.json — ${e.message}`;
    return;
  }

  if (!Array.isArray(history) || history.length === 0) {
    statusEl.textContent = "No benchmark runs yet.";
    return;
  }

  history.sort((a, b) => (a.timestamp < b.timestamp ? -1 : 1));

  const n = history.length;
  metaEl.textContent = `${n} commit${n !== 1 ? "s" : ""}`;

  const labels = history.map((h) => {
    if (h.hash && h.hash !== "unknown") return h.hash.slice(0, 7);
    return new Date(h.timestamp).toLocaleDateString();
  });

  const groupMeta = new Map();
  for (const run of history) {
    for (const m of run.measurements || []) {
      const k = m.group_key || "";
      if (!groupMeta.has(k)) groupMeta.set(k, new Set());
      groupMeta.get(k).add(m.series_key);
    }
  }

  const sortedKeys = [...groupMeta.keys()].sort();

  statusEl.style.display = "none";
  gridEl.style.display = "";

  for (const gk of sortedKeys) {
    const seriesKeys = [...groupMeta.get(gk)].sort();

    const series = Object.fromEntries(
      seriesKeys.map((sk) => [sk, Array(n).fill(null)]),
    );
    history.forEach((run, i) => {
      for (const m of run.measurements || []) {
        if ((m.group_key || "") === gk && series[m.series_key] !== undefined) {
          series[m.series_key][i] = durationToMs(m.duration);
        }
      }
    });

    const card = document.createElement("div");
    card.className = "card";
    const title = document.createElement("div");
    title.className = "card-title";
    const wrap = document.createElement("div");
    wrap.className = "chart-wrap";
    const canvas = document.createElement("canvas");

    title.textContent = meta.groups?.[gk]?.title ?? gk;
    wrap.appendChild(canvas);
    card.appendChild(title);
    card.appendChild(wrap);
    gridEl.appendChild(card);

    new Chart(canvas, {
      type: "line",
      data: {
        labels,
        datasets: seriesKeys.map((sk) => {
          const sm = meta.series?.[sk] ?? {};
          const color = sm.color ?? colorFor(sk);
          return {
            label: sm.label ?? sk,
            data: series[sk],
            borderColor: color,
            backgroundColor: color + "28",
            borderWidth: 2,
            pointRadius: n <= 30 ? 4 : 2,
            pointHoverRadius: 6,
            tension: 0.3,
            spanGaps: true,
          };
        }),
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        interaction: { mode: "index", intersect: false },
        plugins: {
          legend: {
            labels: { color: "#8b949e", boxWidth: 14, padding: 16 },
          },
          tooltip: {
            backgroundColor: "#1c2128",
            borderColor: "#30363d",
            borderWidth: 1,
            titleColor: "#e6edf3",
            bodyColor: "#8b949e",
            callbacks: {
              title: (items) => {
                const run = history[items[0].dataIndex];
                const dt = new Date(run.timestamp).toLocaleString();
                const h =
                  run.hash && run.hash !== "unknown"
                    ? ` (${run.hash.slice(0, 7)})`
                    : "";
                return dt + h;
              },
              label: (item) => {
                const ms = item.raw;
                const val = ms !== null ? formatMs(ms) : "no data";
                return `  ${item.dataset.label}:  ${val}`;
              },
            },
          },
        },
        scales: {
          x: {
            ticks: {
              color: "#8b949e",
              maxRotation: 45,
              autoSkip: true,
              maxTicksLimit: 20,
            },
            grid: { color: "#21262d" },
          },
          y: {
            beginAtZero: true,
            ticks: {
              color: "#8b949e",
              callback: (v) => formatMs(v),
            },
            grid: { color: "#21262d" },
          },
        },
      },
    });
  }
})();
