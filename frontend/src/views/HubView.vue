<template>
  <div class="container">
    <p class="subtitle">Pick a game to play with friends. Everyone who opens the same link joins the same room.</p>

    <div class="games-grid">
      <router-link to="/hotpot" class="game-card">
        <div class="card-icon"><Soup :size="36" /></div>
        <div class="card-title">Hotpot Night</div>
        <div class="card-desc">Cook together at a virtual table. Drop in ingredients, cheer, and chat while the broth simmers.</div>
        <div class="card-badge"><span class="dot"></span> Multiplayer</div>
        <div class="card-btn">Sit down →</div>
      </router-link>

      <router-link to="/exploding-kitchen" class="game-card">
        <div class="card-icon"><Bomb :size="36" /></div>
        <div class="card-title">Exploding Kitchen</div>
        <div class="card-desc">A card game of cooking chaos. Draw cards, avoid the exploding kitchen, use tactics to be the last chef standing.</div>
        <div class="card-badge"><span class="dot"></span> 2–6 Players</div>
        <div class="card-btn">Start cooking →</div>
      </router-link>

      <div class="coming-soon">
        <div class="card-icon"><Target :size="36" /></div>
        <div class="card-title">Coming Soon</div>
        <div class="card-desc">More games are on the way. Stay tuned for new ways to hang out with friends.</div>
        <div class="card-btn secondary" disabled>Not yet available</div>
      </div>
    </div>

    <footer>
      Built with warmth · <a href="https://github.com">Ping</a>
    </footer>
  </div>
</template>

<script setup>
import { Soup, Bomb, Target } from '@lucide/vue'
</script>

<style scoped>
.container {
  max-width: 900px;
  margin: 0 auto;
  padding: 40px 24px 80px;
}

.subtitle {
  color: var(--mild-cream);
  opacity: 0.7;
  font-size: 1.05rem;
  line-height: 1.6;
  max-width: 480px;
  margin: 0 auto 40px;
  text-align: center;
}

.games-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 24px;
  margin-bottom: 60px;
}

.game-card {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 20px;
  padding: 32px 28px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  transition: transform 0.2s ease, border-color 0.2s ease;
  position: relative;
  overflow: hidden;
}

.game-card:hover {
  transform: translateY(-4px);
  border-color: var(--chili-orange);
}

.card-icon {
  font-size: 2.6rem;
  line-height: 1;
  display: flex;
  align-items: center;
  color: var(--chili-orange);
}

.card-title {
  font-family: 'Baloo 2', sans-serif;
  font-weight: 700;
  font-size: 1.35rem;
}

.card-desc {
  color: var(--mild-cream);
  opacity: 0.75;
  font-size: 0.92rem;
  line-height: 1.55;
  flex: 1;
}

.card-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.76rem;
  padding: 5px 12px;
  border-radius: 100px;
  width: fit-content;
  background: rgba(238, 194, 92, 0.1);
  color: var(--gold);
  border: 1px solid rgba(238, 194, 92, 0.2);
}

.card-badge .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--gold);
}

.card-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 24px;
  border-radius: 100px;
  font-weight: 600;
  font-size: 0.9rem;
  border: none;
  background: linear-gradient(135deg, var(--chili-orange), var(--broth-red));
  color: var(--steam-cream);
  transition: transform 0.15s ease;
}

.card-btn:hover {
  transform: translateY(-2px);
}

.card-btn.secondary {
  background: transparent;
  border: 1px solid var(--line);
  color: var(--mild-cream);
}

.coming-soon {
  background: var(--panel);
  border: 1px dashed rgba(253, 243, 228, 0.1);
  border-radius: 20px;
  padding: 32px 28px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  opacity: 0.4;
}

.coming-soon .card-icon {
  filter: grayscale(1);
}

footer {
  text-align: center;
  color: var(--mild-cream);
  opacity: 0.4;
  font-size: 0.8rem;
  border-top: 1px solid var(--line);
  padding-top: 28px;
}

footer a {
  color: var(--gold);
  text-decoration: none;
}

@media (max-width: 600px) {
  .container {
    padding: 40px 16px 60px;
  }
  h1 {
    font-size: 1.8rem;
  }
  .games-grid {
    grid-template-columns: 1fr;
  }
}
</style>
