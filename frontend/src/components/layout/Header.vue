<template>
    <div id="cb-header" class="header-container" hx-preserve>
        <div id="hgradient"></div>
        <div class="header-content">
            <!-- Save/Restore Controls on left -->
            <div class="left-controls">
                <SaveRestoreControls @save="$emit('save')" @restore="$emit('restore', $event)" />
            </div>
            <!-- Manifold Title Logo in center -->
            <div class="box-container">
                Manifold 
            </div>
            <!-- Empty div for right side balance -->
            <div class="right-controls"></div>
        </div>
    </div>
</template>

<script setup>
import SaveRestoreControls from '../layout/SaveRestoreControls.vue';

defineEmits(['save', 'restore']);
</script>

<style scoped>
#cb-header {
    position: relative;
    height: 50px;
    background-color: #222;
}

.header-content {
    display: flex;
    justify-content: space-between;
    align-items: center;
    height: 100%;
    padding: 0 20px;
}

.left-controls, .right-controls {
    display: flex;
    align-items: center;
    width: 150px; /* Ensure equal width for left and right controls */
}

.box-container {
    /* Roboto Font */
    font-family: 'Roboto', sans-serif;
    font-size: 24px;
    font-weight: 700;
    color: white;
    text-align: center; /* Center text within the box */
    flex-grow: 1; /* Allow the center box to grow and take up remaining space */
}
</style>