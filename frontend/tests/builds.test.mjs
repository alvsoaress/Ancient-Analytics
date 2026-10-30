import test from 'node:test'
import assert from 'node:assert/strict'
import { formatPurchaseTime, sampleExclusions } from '../src/builds.ts'

test('tempo de compra trata mediana fracionada, horas e compras anteriores ao início', () => {
  assert.equal(formatPurchaseTime(0), 'Antes do início')
  assert.equal(formatPurchaseTime(-30), 'Antes do início')
  assert.equal(formatPurchaseTime(900), '15:00')
  assert.equal(formatPurchaseTime(959.5), '16:00')
  assert.equal(formatPurchaseTime(3661), '61:01')
})
test('resumo de exclusões contabiliza todos os motivos sem contar dados disponíveis', () => {
  assert.equal(sampleExclusions({ matchesUnavailable: 2, matchesWithoutPurchaseLog: 3, matchesOutsidePosition: 1, matchesWithoutPosition: 2, matchesWithUnknownItems: 1, matchesWithoutCoreItems: 1 }), 10)
  assert.equal(sampleExclusions({ matchesUnavailable: 0, matchesWithoutPurchaseLog: 0, matchesOutsidePosition: 0, matchesWithoutPosition: 0, matchesWithUnknownItems: 0, matchesWithoutCoreItems: 0 }), 0)
})

test('talentos não exibem marcadores do catálogo nem inventam seus valores', async () => {
  const { talentText } = await import('../src/builds.ts')
  assert.deepEqual(talentText('+{s:bonus_bonus_damage} Shuriken Toss Damage'), { label: 'Shuriken Toss Damage', hasMissingValue: true })
  assert.deepEqual(talentText('-{s:bonus_damage_reduction_pct}% Damage Taken in Shadow Walk'), { label: 'Damage Taken in Shadow Walk', hasMissingValue: true })
  assert.deepEqual(talentText('+30 Damage'), { label: '+30 Damage', hasMissingValue: false })
  assert.deepEqual(talentText(null), { label: null, hasMissingValue: false })
})
