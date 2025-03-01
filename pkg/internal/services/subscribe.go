package services

import "sync"

// ChannelID -> UserID -> Client ID
var subscribeInfo = make(map[uint]map[uint]string)
var subscribeLock sync.Mutex

// If user subscribed to a channel
// Push the new message to them via websocket
// And skip the notification

func CheckSubscribed(UserID uint, ChannelID uint) bool {
	if _, ok := subscribeInfo[ChannelID]; ok {
		if _, ok := subscribeInfo[ChannelID][UserID]; ok {
			return true
		}
	}
	return false
}

func SubscribeChannel(userId uint, channelId uint, clientId string) {
	subscribeLock.Lock()
	defer subscribeLock.Unlock()
	if _, ok := subscribeInfo[channelId]; !ok {
		subscribeInfo[channelId] = make(map[uint]string)
	}
	subscribeInfo[channelId][userId] = clientId
}

func UnsubscribeChannel(userId uint, channelId uint) {
	subscribeLock.Lock()
	defer subscribeLock.Unlock()
	if _, ok := subscribeInfo[channelId]; ok {
		delete(subscribeInfo[channelId], userId)
	}
}

func UnsubscribeAll(userId uint) {
	subscribeLock.Lock()
	defer subscribeLock.Unlock()
	for _, v := range subscribeInfo {
		delete(v, userId)
	}
}

func UnsubscribeAllWithChannels(channelId uint) {
	subscribeLock.Lock()
	defer subscribeLock.Unlock()
	delete(subscribeInfo, channelId)
}

func UnsubscribeAllWithClient(clientId string) {
	subscribeLock.Lock()
	defer subscribeLock.Unlock()
	for _, v := range subscribeInfo {
		for k, item := range v {
			if item == clientId {
				delete(v, k)
			}
		}
	}
}
