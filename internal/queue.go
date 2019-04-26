/**
*    Copyright (C) 2019-present C2CV Holdings, LLC.
*
*    This program is free software: you can redistribute it and/or modify
*    it under the terms of the Server Side Public License, version 1,
*    as published by C2CV Holdings, LLC.
*
*    This program is distributed in the hope that it will be useful,
*    but WITHOUT ANY WARRANTY; without even the implied warranty of
*    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*    Server Side Public License for more details.
*
*    You should have received a copy of the Server Side Public License
*    along with this program. If not, see
*    <http://www.mongodb.com/licensing/server-side-public-license>.
*
*    As a special exception, the copyright holders give permission to link the
*    code of portions of this program with the OpenSSL library under certain
*    conditions as described in each individual source file and distribute
*    linked combinations including the program with the OpenSSL library. You
*    must comply with the Server Side Public License in all respects for
*    all of the code used other than as permitted herein. If you modify file(s)
*    with this exception, you may extend this exception to your version of the
*    file(s), but you are not obligated to do so. If you do not wish to do so,
*    delete this exception statement from your version. If you delete this
*    exception statement from all source files in the program, then also delete
*    it in the license file.
*/

package internal

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"log"
	"sync"
)

// Queue is a basic FIFO queue based on a circular list that resizes as needed.
type Queue struct {
	nodes []*chainhash.Hash
	size  int
	head  int
	tail  int
	count int
	mux sync.Mutex
}

func NewQueue(size int) *Queue {
	return &Queue{
		nodes: make([]*chainhash.Hash, size),
		size:  size,
	}
}

// Push adds a node to the queue.
func (q *Queue) Push(n *chainhash.Hash) {
	q.mux.Lock()

	defer q.mux.Unlock()

	//see if n is in the queue (inefficient but good enough for now)
	for _, node := range q.nodes {
		if node != nil && node.String() == n.String() {
			log.Println("Duplicate hash found - all is well.")
			return //skip a hash that we already have
		}
	}

	if q.head == q.tail && q.count > 0 {
		nodes := make([]*chainhash.Hash, len(q.nodes)+q.size)
		copy(nodes, q.nodes[q.head:])
		copy(nodes[len(q.nodes)-q.head:], q.nodes[:q.head])
		q.head = 0
		q.tail = len(q.nodes)
		q.nodes = nodes
	}
	q.nodes[q.tail] = n
	q.tail = (q.tail + 1) % len(q.nodes)
	q.count++
}

// Pop removes and returns a node from the queue in first to last order.
func (q *Queue) Pop() *chainhash.Hash {
	q.mux.Lock()

	defer q.mux.Unlock()

	if q.count == 0 {
		return nil
	}
	node := q.nodes[q.head]
	q.head = (q.head + 1) % len(q.nodes)
	q.count--
	return node
}

func (q *Queue) Peek() *chainhash.Hash {
	q.mux.Lock()

	defer q.mux.Unlock()

	if q.count == 0 {
		return nil
	}
	return q.nodes[q.head]
}

func (q *Queue) Len() int {
	q.mux.Lock()

	defer q.mux.Unlock()

	return q.count
}