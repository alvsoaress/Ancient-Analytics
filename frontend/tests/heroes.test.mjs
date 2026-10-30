import test from 'node:test'
import assert from 'node:assert/strict'
import { findHeroes, heroBuildUrl, parseHeroId, parseBuildPosition } from '../src/heroes.ts'

const heroes = [
  { id: 62, localized_name: 'Bounty Hunter', primary_attr: 'agi' },
  { id: 1, localized_name: 'Anti-Mage', primary_attr: 'agi' },
  { id: 74, localized_name: 'Invóker', primary_attr: 'int' },
]
test('busca ignora caixa, espaços, hífens e acentos sem alterar o catálogo', () => {
  assert.deepEqual(findHeroes(heroes, ' ANTI MAGE ').map(h => h.id), [1])
  assert.deepEqual(findHeroes(heroes, 'invoker').map(h => h.id), [74])
  assert.deepEqual(findHeroes(heroes, 'hunter').map(h => h.id), [62])
  assert.deepEqual(findHeroes(heroes, 'herói inexistente'), [])
  assert.deepEqual(findHeroes(heroes, '', 'int').map(h => h.id), [74])
  assert.deepEqual(heroes.map(h => h.id), [62, 1, 74])
})
test('identificadores seguem o limite do backend e rejeitam URLs inválidas', () => {
  for (const value of [null, '', '0', '-1', '1.5', 'abc', '62xyz', '2147483648', '1e2']) assert.equal(parseHeroId(value), null)
  assert.equal(parseHeroId('62'), 62)
  assert.equal(parseHeroId('2147483647'), 2147483647)
})
test('posição e endereço permitem compartilhar um herói com seu filtro', () => {
  assert.equal(parseBuildPosition(null), '')
  assert.equal(parseBuildPosition('support4'), 'support4')
  assert.equal(parseBuildPosition(''), null)
  assert.equal(parseBuildPosition('suporte'), null)
  assert.equal(heroBuildUrl(62), '/?heroi=62')
  assert.equal(heroBuildUrl(62, 'support4'), '/?heroi=62&posicao=support4')
})
